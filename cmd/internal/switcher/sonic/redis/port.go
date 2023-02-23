package redis

import (
	"context"
	"fmt"
	"github.com/vishvananda/netlink"
)

func (a *Applier) ensurePortMTU(ctx context.Context, interfaceName string, mtu int, isFEC bool) error {
	fromRedis, err := a.db.Config.GetPortMTU(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve port info for %s from redis: %w", interfaceName, err)
	}

	fromSys, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve port info for %s via netlink: %w", interfaceName, err)
	}

	if fromRedis != fromSys.Attrs().MTU {
		return fmt.Errorf("different port MTU in redis %v and reported by netlink %v for interface %s", fromRedis, fromSys, interfaceName)
	}

	if fromRedis == mtu {
		return nil
	}

	a.log.Infof("update port info for %s", interfaceName)
	return a.db.Config.SetPort(ctx, interfaceName, mtu, isFEC)
}
