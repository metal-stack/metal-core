package redis

import (
	"context"
	"fmt"
)

func (a *Applier) ensurePortConfiguration(ctx context.Context, portName, mtu string, isUp bool) error {
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

	if p.AdminStatus != isUp {
		a.log.Debug("set admin status to", "port", portName, "admin_status_up", isUp)
		return a.db.Config.SetAdminStatusUp(ctx, portName, isUp)
	}

	return nil
}
