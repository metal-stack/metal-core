package redis

import (
	"context"
	"fmt"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func (a *Applier) ensurePortConfiguration(ctx context.Context, portName, mtu string, adminStatus types.PortStatus) error {
	p, err := a.db.Config.GetPort(ctx, portName)
	if err != nil {
		return fmt.Errorf("could not retrieve port info for %s from redis: %w", portName, err)
	}
	if p == nil {
		return fmt.Errorf("port %s does not exist in CONFIG_DB", portName)
	}

	if p.Mtu != mtu {
		a.log.Debug("set port mtu to", "port", portName, "mtu", mtu)
		err = a.db.Config.SetPortMtu(ctx, portName, mtu)
		if err != nil {
			return err
		}
	}

	if adminStatus == "" {
		return nil
	}

	if p.AdminStatus != string(adminStatus) {
		a.log.Debug("set admin status to", "port", portName, "admin_status", adminStatus)
		if err := a.db.Config.SetAdminStatus(ctx, portName, adminStatus); err != nil {
			return err
		}
	}

	// also if the CONFIG_DB already has it, so a port whose admin status never got applied is reported
	return a.waitFor(ctx, a.portState.HasAdminStatus(portName, adminStatus)...)
}
