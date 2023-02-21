package redis

import (
	"context"
	"fmt"

	"github.com/vishvananda/netlink"
)

func (a *Applier) ensureInterfaceIsVrfMember(ctx context.Context, interfaceName, vrf string) error {
	fromRedis, err := a.c.getVrfMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vrf membership for %s from redis: %w", interfaceName, err)
	}

	fromSys, err := getVrfMembership(interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vrf membership for %s via netlink: %w", interfaceName, err)
	}

	if fromRedis != fromSys {
		return fmt.Errorf("different state in redis %s and reported by netlink %v for interface %s", fromRedis, fromSys, interfaceName)
	}

	if fromRedis == vrf {
		return nil
	} else if len(fromRedis) != 0 {
		return fmt.Errorf("interface %s already member of a different vrf %v", interfaceName, fromRedis)
	}

	a.log.Infof("add interface %s to vrf %s", interfaceName, vrf)
	return a.c.setVrfMember(ctx, interfaceName, vrf)
}

func getVrfMembership(interfaceName string) (string, error) {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return "", fmt.Errorf("unable to get kernel info of the interface %s: %w", interfaceName, err)
	}

	if link.Attrs().MasterIndex == 0 {
		return "", nil
	}

	master, err := netlink.LinkByIndex(link.Attrs().MasterIndex)
	if err != nil {
		return "", fmt.Errorf("unable to get the master of the interface %s: %w", interfaceName, err)
	}
	if master.Type() == "vrf" {
		return master.Attrs().Name, nil
	}
	return "", nil
}
