package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
)

func (a *Applier) ensureNotRouted(ctx context.Context, interfaceName string) error {
	oid, ok := a.rifOidMap[interfaceName]
	if !ok {
		return fmt.Errorf("no mapping of router interface %s to OID", interfaceName)
	}
	routed, err := a.db.Asic.ExistRouterInterface(ctx, oid)
	if err != nil {
		return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
	}
	if !routed {
		return nil
	}

	a.log.Infof("remove routing configuration for interface %s", interfaceName)
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
				a.log.Debugf("interface %s is still routed", interfaceName)
				return fmt.Errorf("interface %s is still routed", interfaceName)
			}
			return nil
		},
	)
}
