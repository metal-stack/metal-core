package redis

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func (a *Applier) ensureInterfaceIsVrfMember(ctx context.Context, interfaceName, vrfName string) error {
	current, err := a.db.Config.GetVrfMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vrfName membership for %s from redis: %w", interfaceName, err)
	}

	if current == vrfName {
		return nil
	} else if len(current) != 0 {
		return fmt.Errorf("interface %s already member of a different vrfName %v", interfaceName, current)
	}

	a.log.Info("add interface to vrfName ", "name", interfaceName, "vrf", vrfName)
	return a.db.Config.SetVrfMember(ctx, interfaceName, vrfName)
}

func (a *Applier) configureVrf(ctx context.Context, vrf *types.Vrf) error {
	exist, err := a.db.Config.ExistVrf(ctx, vrf.Name)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVrf(ctx, vrf.Name, vrf.VNI); err != nil {
			return fmt.Errorf("could not create vrf %s: %w", vrf.Name, err)
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
		if err := a.db.Config.CreateVlanInterface(ctx, vrf.VLANID, vrf.Name); err != nil {
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

		if slices.Contains(cfg.FRRConfig.VRFs, vrfName) {
			continue
		}

		vni, err := strconv.ParseUint(vrfName[3:], 10, 32)
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
