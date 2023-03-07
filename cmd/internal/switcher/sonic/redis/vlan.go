package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
	"github.com/vishvananda/netlink"
)

func (a *Applier) ensureNotBridged(ctx context.Context, interfaceName string) error {
	oid, ok := a.bridgePortOidMap[interfaceName]
	if !ok {
		return nil
	}
	bridged, err := a.db.Asic.ExistBridgePort(ctx, oid)
	if err != nil {
		return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
	}
	if !bridged {
		return nil
	}

	vlans, err := a.db.Config.GetVlanMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan membership for %s from Redis: %w", interfaceName, err)
	}

	for _, vlan := range vlans {
		a.log.Infof("remove interface %s from vlan %s", interfaceName, vlan)
		err := a.db.Config.DeleteVlanMember(ctx, interfaceName, vlan)
		if err != nil {
			return fmt.Errorf("could not remove interface %s from vlan %s", interfaceName, vlan)
		}
	}

	return retry.Do(
		func() error {
			bridged, err := a.db.Asic.ExistBridgePort(ctx, oid)
			if err != nil {
				return err
			}
			if bridged {
				return fmt.Errorf("interface %s still bridged", interfaceName)
			}
			return nil
		},
	)
}

func (a *Applier) ensureInterfaceIsVlanMember(ctx context.Context, interfaceName, vlan string) error {
	fromRedis, err := a.db.Config.GetVlanMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan membership for %s from redis: %w", interfaceName, err)
	}

	fromSys, err := getVlanMembership(interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan membership for %s via netlink: %w", interfaceName, err)
	}

	// an interface should belong to at most one VLAN and therefore no sorting necessary
	if !equal(fromRedis, fromSys) {
		return fmt.Errorf("different state in redis %v and reported by netlink %v for interface %s", fromRedis, fromSys, interfaceName)
	}

	if len(fromRedis) == 1 && fromRedis[0] == vlan {
		return nil
	} else if len(fromRedis) != 0 {
		return fmt.Errorf("interface %s already member of a different vlan %v", interfaceName, fromRedis)
	}

	a.log.Infof("add interface %s to vlan %s", interfaceName, vlan)
	return a.db.Config.SetVlanMember(ctx, interfaceName, vlan)
}

func getVlanMembership(interfaceName string) ([]string, error) {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("unable to get kernel info of interface:%s %w", interfaceName, err)
	}
	vlansByInterface, err := netlink.BridgeVlanList()
	if err != nil {
		return nil, fmt.Errorf("unable to get kernel info of vlans %w", err)
	}
	vlanInfos, ok := vlansByInterface[int32(link.Attrs().Index)]
	if !ok {
		return nil, nil
	}

	vlans := make([]string, 0, len(vlanInfos))
	for _, vlanInfo := range vlanInfos {
		vlans = append(vlans, fmt.Sprintf("Vlan%d", vlanInfo.Vid))
	}
	return vlans, nil
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
