package redis

import (
	"context"
	"fmt"
)

func (a *Applier) ensurePortConfiguration(ctx context.Context, portName, mtu string) error {
	p, err := a.db.Config.GetPort(ctx, portName)
	if err != nil {
		return fmt.Errorf("could not retrieve port info for %s from redis: %w", portName, err)
	}

	if p.Mtu != mtu {
		a.log.Debug("set port mtu to", "port", portName, "mtu", mtu)
		err = a.db.Config.SetPortMtu(ctx, portName, mtu)
		if err != nil {
			return err
		}
	}

	if !p.AdminStatus {
		a.log.Debug("set admin status to", "port", portName, "admin_status", "up")
		return a.db.Config.SetAdminStatusUp(ctx, portName)
	}

	return nil
}
