package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
)

func (a *Applier) ensurePortConfiguration(ctx context.Context, portName, mtu string, isFecRs bool) error {
	p, err := a.db.Config.GetPort(ctx, portName)
	if err != nil {
		return fmt.Errorf("could not retrieve port info for %s from redis: %w", portName, err)
	}

	if p.FecRs != isFecRs {
		a.log.Debug("set port rs mode to", "port", portName, "mode", isFecRs)
		err = a.ensurePortFecMode(ctx, portName, isFecRs)
		if err != nil {
			return err
		}
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

func (a *Applier) ensurePortFecMode(ctx context.Context, portName string, wantFecRs bool) error {
	err := a.db.Config.SetPortFecMode(ctx, portName, wantFecRs)
	if err != nil {
		return fmt.Errorf("could not update Fec for port %s: %w", portName, err)
	}

	oid, ok := a.portOidMap[portName]
	if !ok {
		return fmt.Errorf("no mapping of port %s to OID", portName)
	}

	return retry.Do(
		func() error {
			isFecRs, err := a.db.Asic.InFecModeRs(ctx, oid)
			if err != nil {
				return err
			}
			if isFecRs != wantFecRs {
				return fmt.Errorf("port %s still has rs mode = %v, but want %v", portName, isFecRs, wantFecRs)
			}
			return nil
		},
	)
}
