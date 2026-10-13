package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	sonic "github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type Applier struct {
	db          *db.DB
	log         *slog.Logger
	previousCfg *types.Conf
	quirks      sonic.Quirks
	portState   sonic.PortStateChecker
}

func NewApplier(log *slog.Logger, sonicDb *db.DB, quirks sonic.Quirks) (*Applier, error) {
	return &Applier{
		db:        sonicDb,
		log:       log,
		quirks:    quirks,
		portState: quirks.PortState(log, sonicDb),
	}, nil
}

func (a *Applier) Apply(ctx context.Context, cfg *types.Conf) error {
	var (
		errs    []error
		changed bool
	)

	if err := a.configureLLDP(ctx, cfg); len(err) != 0 {
		errs = append(errs, err...)
	}

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

	if err := a.portState.Refresh(ctx); err != nil {
		return err
	}

	a.log.Debug("configure underlay ports", "ports", cfg.Ports.Underlay)
	for _, interfaceName := range cfg.Ports.Underlay {
		if err := a.configureUnderlayPort(ctx, interfaceName, cfg.Ports.AdminStatus[interfaceName]); err != nil {
			errs = append(errs, err)
		}
	}

	a.log.Debug("configure unprovisioned ports", "ports", cfg.Ports.Unprovisioned)
	for _, interfaceName := range cfg.Ports.Unprovisioned {
		pxeVlan := fmt.Sprintf("Vlan%d", cfg.PXEVlanID)
		if err := a.configureUnprovisionedPort(ctx, interfaceName, cfg.Ports.AdminStatus[interfaceName], pxeVlan); err != nil {
			errs = append(errs, err)
		}
	}

	a.log.Debug("configure firewall ports", "ports", cfg.Ports.Firewalls)
	for interfaceName := range cfg.Ports.Firewalls {
		if err := a.configureFirewallPort(ctx, interfaceName, cfg.Ports.AdminStatus[interfaceName]); err != nil {
			errs = append(errs, err)
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
			}
		}
	}

	err = a.cleanupVrfs(ctx, cfg)
	if err != nil {
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

func (a *Applier) configureLLDP(ctx context.Context, cfg *types.Conf) []error {
	var errs []error

	err := a.db.Config.SetLldpTxInterval(ctx, a.quirks.LldpTxIntervalMode())
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to update LLDP tx_interval: %w", err))
	}
	ports, err := a.GetPorts(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get ports: %w", err))
	}
	for _, port := range ports {
		err := a.db.Config.SetLldpForPor(ctx, port.Name, port.Alias, a.quirks.LldpPortConfigMode())
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update LLDP settings for port %s: %w", port.Name, err))
		}
	}

	return errs
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
	vtep, err := a.db.Config.GetVTEPName(ctx)
	if err != nil {
		return err
	}

	for _, e := range a.quirks.VrfEntries(vrfName, vrf, vtep) {
		if err := a.db.Config.EnsureEntry(ctx, e); err != nil {
			return fmt.Errorf("could not configure %v for vrf %s: %w", e.Key, vrfName, err)
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

		vni, err := strconv.ParseUint(strings.TrimPrefix(vrfName, "Vrf"), 10, 32)
		if err != nil {
			return fmt.Errorf("could not parse vni for vrf %s: %w", vrfName, err)
		}

		vrf := &types.Vrf{
			VNI: uint32(vni),
		}

		a.log.Debug("find vxlan tunnel map for vrf", "vrf", vrf)
		tunnelMap, err := a.db.Config.FindVxlanTunnelMapByVni(ctx, uint32(vni))
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
	vtep, err := a.db.Config.GetVTEPName(ctx)
	if err != nil {
		return err
	}

	for _, e := range slices.Backward(a.quirks.VrfEntries(vrfName, vrf, vtep)) {
		deleted, err := a.db.Config.DeleteEntry(ctx, e.Key)
		if err != nil {
			return fmt.Errorf("could not delete %v of vrf %s: %w", e.Key, vrfName, err)
		}
		if deleted {
			a.log.Debug("deleted vrf entry", "vrf", vrfName, "key", e.Key)
		}
	}
	return nil
}
