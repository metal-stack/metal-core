package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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

func (a *Applier) Apply(ctx context.Context, cfg *types.Conf) error {
	var (
		errs    []error
		changed bool
	)

	if a.previousCfg != nil {
		diff := cmp.Diff(a.previousCfg, cfg)
		if diff != "" {
			changed = true
			a.log.Debug("interface changes", "changes", diff)
		}
	}

	currentPorts, err := a.db.Config.GetPorts(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	for _, port := range currentPorts {
		if types.PortStatus(port.AdminStatus) != cfg.Ports.AdminStatus[port.Name] {
			changed = true
			break
		}
	}

	if !changed {
		a.log.Info("no changes on interfaces detected, nothing to do")
		return nil
	}

	if err := a.refreshOidMaps(ctx); err != nil {
		return err
	}

	a.log.Debug("configure underlay ports", "ports", cfg.Ports.Underlay)
	for _, interfaceName := range cfg.Ports.Underlay {
		if err := a.configureUnderlayPort(ctx, interfaceName, cfg.Ports.AdminStatus[interfaceName]); err != nil {
			errs = append(errs, err)
		}
	}

	// ports whose transition failed keep their boot ACL binding; they might still be routed in the boot VRF
	failed := map[string]bool{}

	// bind the boot ACLs to current and desired ports before moving any port, so that no port
	// is routed in the boot VRF without the filter. Unbind ports after they leave the boot VRF.
	// If the ACL tables are missing in L3 mode, nothing is touched.
	bound, err := a.bootACLBoundPorts(ctx)
	if err != nil {
		return err
	}
	if cfg.Boot != nil {
		if err := a.ensureBootACLBinding(ctx, union(bound, cfg.Ports.Unprovisioned), true); err != nil {
			return err
		}

		a.log.Debug("configure boot vrf", "vrf", cfg.Boot.Vrf, "vni", cfg.Boot.VNI)
		if err := a.configureVrf(ctx, cfg.Boot.Vrf, &types.Vrf{VNI: cfg.Boot.VNI, VLANID: cfg.Boot.VLANID}); err != nil {
			errs = append(errs, err)
		}

		a.log.Debug("configure unprovisioned ports in boot vrf", "ports", cfg.Ports.Unprovisioned)
		for _, interfaceName := range cfg.Ports.Unprovisioned {
			bootPort, ok := cfg.Boot.Ports[interfaceName]
			if !ok {
				errs = append(errs, fmt.Errorf("no boot prefix for unprovisioned port %s", interfaceName))
				failed[interfaceName] = true
				continue
			}
			if err := a.configureBootPort(ctx, interfaceName, cfg.Ports.AdminStatus[interfaceName], cfg.Boot.Vrf, bootPort); err != nil {
				errs = append(errs, err)
				failed[interfaceName] = true
			}
		}
	} else {
		a.log.Debug("configure unprovisioned ports", "ports", cfg.Ports.Unprovisioned)
		for _, interfaceName := range cfg.Ports.Unprovisioned {
			pxeVlan := fmt.Sprintf("Vlan%d", cfg.PXEVlanID)
			if err := a.configureUnprovisionedPort(ctx, interfaceName, cfg.Ports.AdminStatus[interfaceName], pxeVlan); err != nil {
				errs = append(errs, err)
				failed[interfaceName] = true
			}
		}
	}

	a.log.Debug("configure firewall ports", "ports", cfg.Ports.Firewalls)
	for interfaceName := range cfg.Ports.Firewalls {
		if err := a.configureFirewallPort(ctx, interfaceName, cfg.Ports.AdminStatus[interfaceName]); err != nil {
			errs = append(errs, err)
			failed[interfaceName] = true
		}
	}

	a.log.Debug("configure port vrfs", "vrfs", cfg.Ports.Vrfs)
	for vrfName, vrf := range cfg.Ports.Vrfs {
		if err := a.configureVrf(ctx, vrfName, vrf); err != nil {
			errs = append(errs, err)
		}
		for _, interfaceName := range vrf.Neighbors {
			if err := a.configureVrfNeighbor(ctx, interfaceName, vrfName, cfg.Ports.AdminStatus[interfaceName]); err != nil {
				errs = append(errs, err)
				failed[interfaceName] = true
			}
		}
	}

	err = a.cleanupVrfs(ctx, cfg)
	if err != nil {
		errs = append(errs, err)
	}

	// unbind ports that successfully left the boot VRF
	var keep []string
	if cfg.Boot != nil {
		keep = union(nil, cfg.Ports.Unprovisioned)
	}
	for _, port := range bound {
		if failed[port] {
			keep = union(keep, []string{port})
		}
	}
	if err := a.ensureBootACLBinding(ctx, keep, cfg.Boot != nil); err != nil {
		errs = append(errs, err)
	}

	// config is only treated as applied if no errors are encountered
	if len(errs) == 0 {
		a.previousCfg = cfg
	}
	return errors.Join(errs...)
}

func (a *Applier) GetPorts(ctx context.Context) ([]*db.Port, error) {
	return a.db.Config.GetPorts(ctx)
}

func (a *Applier) refreshRifOidMap(ctx context.Context) error {
	oidMap, err := a.db.Counters.GetRifNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update rif to oid map: %w", err)
	}
	a.log.Debug("set rif oid map", "map", oidMap)
	a.rifOidMap = oidMap
	return nil
}

func (a *Applier) refreshOidMaps(ctx context.Context) error {
	a.log.Debug("refresh oid maps")

	oidMap, err := a.db.Counters.GetPortNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update port to oid map: %w", err)
	}
	a.log.Debug("set port oid map", "map", oidMap)
	a.portOidMap = oidMap

	oidMap, err = a.db.Counters.GetRifNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update rif to oid ma: %w", err)
	}
	a.log.Debug("set rif oid map", "map", oidMap)
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
	a.log.Debug("set bridge port oid map", "map", oidMap)
	a.bridgePortOidMap = oidMap

	return nil
}

func (a *Applier) configureUnprovisionedPort(ctx context.Context, interfaceName string, adminStatus types.PortStatus, pxeVlan string) error {
	err := a.ensureNotRouted(ctx, interfaceName)
	if err != nil {
		return err
	}

	if err := a.ensurePortConfiguration(ctx, interfaceName, "9000", adminStatus); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureInterfaceIsVlanMember(ctx, interfaceName, pxeVlan)
}

func (a *Applier) configureFirewallPort(ctx context.Context, interfaceName string, adminStatus types.PortStatus) error {
	err := a.ensureNotBridged(ctx, interfaceName)
	if err != nil {
		return err
	}

	// a firewall port lives in the default VRF; remove it from the boot VRF if it was unprovisioned before
	err = a.ensureNotVrfMember(ctx, interfaceName)
	if err != nil {
		return err
	}

	// a firewall port should always be up
	if err := a.ensurePortConfiguration(ctx, interfaceName, "9216", adminStatus); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureUnderlayPort(ctx context.Context, interfaceName string, adminStatus types.PortStatus) error {
	if err := a.ensurePortConfiguration(ctx, interfaceName, "9216", adminStatus); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}
	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureVrfNeighbor(ctx context.Context, interfaceName, vrfName string, adminStatus types.PortStatus) error {
	err := a.ensureNotBridged(ctx, interfaceName)
	if err != nil {
		return err
	}

	err = a.ensureInterfaceIsVrfMember(ctx, interfaceName, vrfName)
	if err != nil {
		return err
	}

	if err := a.ensurePortConfiguration(ctx, interfaceName, "9000", adminStatus); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureVrf(ctx context.Context, vrfName string, vrf *types.Vrf) error {
	exist, err := a.db.Config.ExistVrf(ctx, vrfName)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVrf(ctx, vrfName, vrf.VNI); err != nil {
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

func (a *Applier) cleanupVrfs(ctx context.Context, cfg *types.Conf) error {
	vrfs, err := a.db.Config.GetVrfs(ctx)
	if err != nil {
		return fmt.Errorf("could not retrieve vrfs: %w", err)
	}

	for _, vrfName := range vrfs {
		if vrfName == "default" {
			continue
		}

		if _, found := cfg.Ports.Vrfs[vrfName]; found {
			continue
		}

		if cfg.Boot != nil && vrfName == cfg.Boot.Vrf {
			continue
		}

		vni, err := a.db.Config.GetVrfVni(ctx, vrfName)
		if err != nil {
			return err
		}

		vrf := &types.Vrf{
			VNI: vni,
		}

		a.log.Debug("find vxlan tunnel map for vrf", "vrf", vrf)
		tunnelMap, err := a.db.Config.FindVxlanTunnelMapByVni(ctx, vni)
		if err != nil {
			return fmt.Errorf("could not look up vxlan tunnel map for vni %d: %w", vni, err)
		}

		if tunnelMap != nil {
			vlan, err := strconv.ParseUint(strings.TrimPrefix(tunnelMap.Vlan, "Vlan"), 10, 16)
			if err != nil {
				return fmt.Errorf("could not parse vlan id %s: %w", tunnelMap.Vlan, err)
			}
			vrf.VLANID = uint16(vlan)
		}

		if err := a.cleanupVrf(ctx, vrfName, vrf); err != nil {
			return err
		}
	}

	return nil
}

func (a *Applier) cleanupVrf(ctx context.Context, vrfName string, vrf *types.Vrf) error {
	a.log.Debug("cleanup unused vrf", "vrf", vrf)
	exists, err := a.db.Config.ExistVxlanTunnelMap(ctx, vrf.VLANID, vrf.VNI)
	if err != nil {
		return err
	}
	if exists {
		a.log.Debug("delete vxlan tunnel map", "vlan id", vrf.VLANID, "vni", vrf.VNI)
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
		a.log.Debug("delete vlan interface", "vlan id", vrf.VLANID)
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
		a.log.Debug("delete neighbor suppression", "vlan id", vrf.VLANID)
		if err := a.db.Config.DeleteNeighborSuppression(ctx, vrf.VLANID); err != nil {
			return fmt.Errorf("could not delete neighbor suppression for vlan %d: %w", vrf.VLANID, err)
		}
	}

	exists, err = a.db.Config.ExistVlan(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if exists {
		a.log.Debug("delete vlan", "vlan id", vrf.VLANID)
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
		a.log.Debug("delete vrf", "vrf", vrfName)
		if err := a.db.Config.DeleteVrf(ctx, vrfName); err != nil {
			return fmt.Errorf("could not delete vrf %s: %w", vrfName, err)
		}
	}
	return nil
}
