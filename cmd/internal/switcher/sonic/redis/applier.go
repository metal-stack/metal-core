package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type Applier struct {
	db          *db.DB
	log         *slog.Logger
	previousCfg *types.Conf

	bridgePortOidMap map[string]db.OID
	portOidMap       map[string]db.OID
	rifOidMap        map[string]db.OID
}

func NewApplier(log *slog.Logger, db *db.DB) *Applier {
	return &Applier{
		db:  db,
		log: log,
	}
}

func (a *Applier) Apply(cfg *types.Conf) error {
	var errs []error

	// only process if changes are detected
	if a.previousCfg != nil {
		diff := cmp.Diff(a.previousCfg, cfg)
		if diff == "" {
			a.log.Info("no changes on interfaces detected, nothing to do")
			return nil
		} else {
			a.log.Debug("interface changes", "changes", diff)
		}
	}

	if err := a.refreshOidMaps(); err != nil {
		return err
	}

	for _, interfaceName := range cfg.Ports.Underlay {
		if err := a.configureUnderlayPort(interfaceName, !cfg.Ports.DownPorts[interfaceName]); err != nil {
			errs = append(errs, err)
		}
	}

	for _, interfaceName := range cfg.Ports.Unprovisioned {
		pxeVlan := fmt.Sprintf("Vlan%d", cfg.PXEVlanID)
		if err := a.configureUnprovisionedPort(interfaceName, !cfg.Ports.DownPorts[interfaceName], pxeVlan); err != nil {
			errs = append(errs, err)
		}
	}

	for interfaceName := range cfg.Ports.Firewalls {
		if err := a.configureFirewallPort(interfaceName, !cfg.Ports.DownPorts[interfaceName]); err != nil {
			errs = append(errs, err)
		}
	}

	for vrfName, vrf := range cfg.Ports.Vrfs {
		if err := a.configureVrf(vrfName, vrf); err != nil {
			errs = append(errs, err)
		}
		for _, interfaceName := range vrf.Neighbors {
			if err := a.configureVrfNeighbor(interfaceName, vrfName, !cfg.Ports.DownPorts[interfaceName]); err != nil {
				errs = append(errs, err)
			}
		}
	}

	err := a.cleanupVrfs(cfg)
	if err != nil {
		errs = append(errs, err)
	}

	// config is only treated as applied if no errors are encountered
	if len(errs) == 0 {
		a.previousCfg = cfg
	}
	return errors.Join(errs...)
}

func (a *Applier) refreshOidMaps() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	oidMap, err := a.db.Counters.GetPortNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update port to oid map: %w", err)
	}
	a.portOidMap = oidMap

	oidMap, err = a.db.Counters.GetRifNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update rif to oid ma: %w", err)
	}
	a.rifOidMap = oidMap

	bridgePortMap, err := a.db.Asic.GetPortIdBridgePortMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update bridge port to oid map: %w", err)
	}
	oidMap = make(map[string]db.OID, len(bridgePortMap))
	for port, oid := range a.portOidMap {
		if bridgePort, ok := bridgePortMap[oid]; ok {
			oidMap[port] = bridgePort
		}
	}
	a.bridgePortOidMap = oidMap

	return nil
}

func (a *Applier) configureUnprovisionedPort(interfaceName string, isUp bool, pxeVlan string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureNotRouted(ctx, interfaceName)
	if err != nil {
		return err
	}

	// unprovisioned ports should be up
	if err := a.ensurePortConfiguration(ctx, interfaceName, "9000", isUp); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureInterfaceIsVlanMember(ctx, interfaceName, pxeVlan)
}

func (a *Applier) configureFirewallPort(interfaceName string, isUp bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureNotBridged(ctx, interfaceName)
	if err != nil {
		return err
	}

	// a firewall port should always be up
	if err := a.ensurePortConfiguration(ctx, interfaceName, "9216", isUp); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureUnderlayPort(interfaceName string, isUp bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// underlay ports should be up
	if err := a.ensurePortConfiguration(ctx, interfaceName, "9216", isUp); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}
	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureVrfNeighbor(interfaceName, vrfName string, isUp bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureNotBridged(ctx, interfaceName)
	if err != nil {
		return err
	}

	err = a.ensureInterfaceIsVrfMember(ctx, interfaceName, vrfName)
	if err != nil {
		return err
	}

	if err := a.ensurePortConfiguration(ctx, interfaceName, "9000", isUp); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureVrf(vrfName string, vrf *types.Vrf) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exist, err := a.db.Config.ExistVrf(ctx, vrfName)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVrf(ctx, vrfName); err != nil {
			return fmt.Errorf("could not create vrf %s: %w", vrfName, err)
		}
	}

	exist, err = a.db.Config.ExistVlan(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVlan(ctx, vrf.VLANID); err != nil {
			return fmt.Errorf("could not create vlan %d: %w", vrf.VLANID, err)
		}
	}

	exist, err = a.db.Config.AreNeighborsSuppressed(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.SuppressNeighbors(ctx, vrf.VLANID); err != nil {
			return fmt.Errorf("could not suppress neighbors for vlan %d: %w", vrf.VLANID, err)
		}
	}

	exist, err = a.db.Config.ExistVlanInterface(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVlanInterface(ctx, vrf.VLANID, vrfName); err != nil {
			return fmt.Errorf("could not create vlan interface for vlan %d: %w", vrf.VLANID, err)
		}
	}

	exist, err = a.db.Config.ExistVxlanTunnelMap(ctx, vrf.VLANID, vrf.VNI)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVxlanTunnelMap(ctx, vrf.VLANID, vrf.VNI); err != nil {
			return fmt.Errorf("could not create vxlan tunnel between vlan %d and vni %d: %w", vrf.VLANID, vrf.VNI, err)
		}
	}

	return nil
}

func (a *Applier) cleanupVrfs(cfg *types.Conf) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vrfs, err := a.db.Config.GetVrfs(ctx)
	if err != nil {
		return fmt.Errorf("could not retrieve vrfs: %w", err)
	}

	for vrfName := range vrfs {
		if _, found := cfg.Ports.Vrfs[vrfName]; found {
			continue
		}

		vni, err := strconv.ParseUint(strings.TrimPrefix(vrfName, "vrf"), 10, 32)
		if err != nil {
			return fmt.Errorf("could not parse vni for vrf %s: %w", vrfName, err)
		}

		vrf := &types.Vrf{
			VNI: uint32(vni),
		}

		tunnelMap, err := a.db.Config.FindVxlanTunnelMapByVni(ctx, uint32(vni))
		if err != nil {
			return fmt.Errorf("could not look up vxlan tunnel map for vni %d: %w", vni, err)
		}

		if tunnelMap != nil {
			vlan, err := strconv.ParseUint(tunnelMap.Vlan, 10, 16)
			if err != nil {
				return fmt.Errorf("could not parse vlan id %s: %w", tunnelMap.Vlan, err)
			}
			vrf.VLANID = uint16(vlan)
		}

		if err := a.cleanupVrf(vrfName, vrf); err != nil {
			return err
		}
	}

	return nil
}

func (a *Applier) cleanupVrf(vrfName string, vrf *types.Vrf) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := a.db.Config.ExistVxlanTunnelMap(ctx, vrf.VLANID, vrf.VNI)
	if err != nil {
		return err
	}
	if exists {
		err = a.db.Config.DeleteVxlanTunnelMap(ctx, vrf.VLANID, vrf.VNI)
		if err != nil {
			return fmt.Errorf("could not remove vxlan tunnel map for vlan %d and vni %d: %w", vrf.VLANID, vrf.VNI, err)
		}
	}

	exists, err = a.db.Config.ExistVlanInterface(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if exists {
		err = a.db.Config.DeleteVlanInterface(ctx, vrf.VLANID)
		if err != nil {
			return fmt.Errorf("could not remove vlan interface %d: %w", vrf.VLANID, err)
		}
	}

	neighborSuppression, err := a.db.Config.AreNeighborsSuppressed(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if neighborSuppression {
		if err := a.db.Config.DeleteNeighborSuppression(ctx, vrf.VLANID); err != nil {
			return fmt.Errorf("could not delete neighbor suppression for vlan %d: %w", vrf.VLANID, err)
		}
	}

	exists, err = a.db.Config.ExistVlan(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if exists {
		err = a.db.Config.DeleteVlan(ctx, vrf.VLANID)
		if err != nil {
			return fmt.Errorf("could not remove vlan %d: %w", vrf.VLANID, err)
		}
	}

	exists, err = a.db.Config.ExistVrf(ctx, vrfName)
	if err != nil {
		return err
	}
	if exists {
		if err := a.db.Config.DeleteVrf(ctx, vrfName); err != nil {
			return fmt.Errorf("could not delete vrf %s: %w", vrfName, err)
		}
	}
	return nil
}
