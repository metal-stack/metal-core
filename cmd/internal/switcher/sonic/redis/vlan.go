package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
	"github.com/vishvananda/netlink"
)

func (a *Applier) ensureInterfaceIsVlanMember(ctx context.Context, interfaceName, vlan string) error {
	fromRedis, err := a.c.getVlanMembership(ctx, interfaceName)
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
	return a.c.setVlanMember(ctx, interfaceName, vlan)
}

func (a *Applier) ensureInterfaceIsNotVlanMember(ctx context.Context, interfaceName string) error {
	fromRedis, err := a.c.getVlanMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan membership for %s from Redis: %w", interfaceName, err)
	}

	fromSys, err := getVlanMembership(interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan membership for %s via netlink: %w", interfaceName, err)
	}

	// an interface should belong to at most one VLAN and therefore no sorting necessary
	if !equal(fromRedis, fromSys) {
		return fmt.Errorf("different state in Redis %v and reported by netlink %v for interface %s", fromRedis, fromSys, interfaceName)
	}

	if len(fromRedis) == 0 {
		return nil
	}

	for _, vlan := range fromRedis {
		a.log.Infof("remove interface %s from vlan %s", interfaceName, vlan)
		err := a.c.deleteVlanMember(ctx, interfaceName, vlan)
		if err != nil {
			return fmt.Errorf("could not remove interface %s from vlan %s", interfaceName, vlan)
		}
	}

	return retry.Do(
		func() error {
			vlans, err := getVlanMembership(interfaceName)
			if err != nil {
				return err
			}
			if len(vlans) != 0 {
				return fmt.Errorf("interface %s still member of vlan %v", interfaceName, vlans)
			}
			return nil
		},
	)
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
