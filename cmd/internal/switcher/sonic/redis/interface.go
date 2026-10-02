package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
)

func (a *Applier) ensureNotRouted(ctx context.Context, interfaceName string) error {
	routed, err := a.portState.isRouted(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
	}
	if !routed {
		return nil
	}

	a.log.Info("remove routing configuration for interface", "name", interfaceName)
	err = a.db.Config.DeleteInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not remove configuration for interface %s: %w", interfaceName, err)
	}

	return retry.Do(
		func() error {
			configured, err := a.portState.isRouted(ctx, interfaceName)
			if err != nil {
				return err
			}
			if configured {
				a.log.Debug("interface is still routed", "name", interfaceName)
				return fmt.Errorf("interface %s is still routed", interfaceName)
			}
			return nil
		},
	)
}
