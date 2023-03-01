package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
)

func (a *Applier) ensurePortConfiguration(ctx context.Context, interfaceName, mtu string, isFecRs bool) error {
	p, err := a.db.Config.GetPort(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve port info for %s from redis: %w", interfaceName, err)
	}

	if p.FecRs != isFecRs {
		a.log.Debugw("set interface %s rs mode to %v", interfaceName, isFecRs)
		err = a.ensurePortFecMode(ctx, interfaceName, isFecRs)
		if err != nil {
			return err
		}
	}

	if p.Mtu != mtu {
		a.log.Debugw("set interface %s mtu to %s", interfaceName, mtu)
		return a.db.Config.SetPortMtu(ctx, interfaceName, mtu)
	}

	return nil
}

func (a *Applier) ensurePortFecMode(ctx context.Context, interfaceName string, wantFecRs bool) error {
	err := a.db.Config.SetPortFecMode(ctx, interfaceName, wantFecRs)
	if err != nil {
		return fmt.Errorf("could not update Fec for interface %s: %w", interfaceName, err)
	}

	oid, ok := a.portOidMap[interfaceName]
	if !ok {
		return fmt.Errorf("no mapping of interface %s to OID", interfaceName)
	}

	return retry.Do(
		func() error {
			isFecRs, err := a.db.Asic.InFecModeRs(ctx, oid)
			if err != nil {
				return err
			}
			if isFecRs != wantFecRs {
				return fmt.Errorf("is interface %s in rs mode = %v, but want %v", interfaceName, isFecRs, wantFecRs)
			}
			return nil
		},
	)
}
