package redis

import (
	"context"
	"fmt"
)

func (a *Applier) ensurePortMTU(ctx context.Context, interfaceName string, mtu int, isFEC bool) error {
	currentMtu, err := a.db.Config.GetPortMTU(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve port info for %s from redis: %w", interfaceName, err)
	}

	if currentMtu == mtu {
		return nil
	}

	a.log.Infof("update port info for %s", interfaceName)
	return a.db.Config.SetPort(ctx, interfaceName, mtu, isFEC)
}
