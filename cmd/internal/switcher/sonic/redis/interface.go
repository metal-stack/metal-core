package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
)

func (a *Applier) ensureInterfaceIsNotConfigured(ctx context.Context, interfaceName string) error {
	configured, err := a.s.existInInterfaceTable(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
	}
	if !configured {
		return nil
	}

	a.log.Infof("remove configuration for interface %s", interfaceName)
	err = a.c.deleteInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not remove configuration for interface %s: %w", interfaceName, err)
	}

	return retry.Do(
		func() error {
			configured, err := a.s.existInInterfaceTable(ctx, interfaceName)
			if err != nil {
				return err
			}
			if configured {
				a.log.Debugf("interface %s is still configured", interfaceName)
				time.Sleep(10 * time.Microsecond)
				return fmt.Errorf("interface %s is still configured", interfaceName)
			}
			return nil
		},
	)
}
