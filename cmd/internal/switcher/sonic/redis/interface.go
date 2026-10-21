package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
)

func (a *Applier) EnsureNotRouted(ctx context.Context, interfaceName string) error {
	oid, ok := a.rifOidMap[interfaceName]
	if !ok {
		return nil
	}
	routed, err := a.db.Asic.ExistRouterInterface(ctx, oid)
	if err != nil {
		return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
	}
	if !routed {
		return nil
	}

	// SONiC defers interface removal until its addresses have been removed.
	if err := a.db.Config.DeleteInterfaceAddresses(ctx, interfaceName); err != nil {
		return fmt.Errorf("could not remove addresses for interface %s: %w", interfaceName, err)
	}

	a.log.Info("remove routing configuration for interface", "name", interfaceName)
	err = a.db.Config.DeleteInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not remove configuration for interface %s: %w", interfaceName, err)
	}

	return retry.Do(
		func() error {
			configured, err := a.db.Asic.ExistRouterInterface(ctx, oid)
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
