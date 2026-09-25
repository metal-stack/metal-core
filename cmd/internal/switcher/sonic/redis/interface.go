package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
)

// ensureNotRouted removes the layer 3 configuration of the interface (vrf membership and addresses)
// and waits until the router interface is gone from the asic.
func (a *Applier) ensureNotRouted(ctx context.Context, interfaceName string) error {
	configured, err := a.db.Config.ExistInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve interface configuration for %s: %w", interfaceName, err)
	}

	var routed bool
	oid, known := a.rifOidMap[interfaceName]
	if known {
		routed, err = a.db.Asic.ExistRouterInterface(ctx, oid)
		if err != nil {
			return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
		}
	}

	if !configured && !routed {
		return nil
	}

	if configured && !known {
		// the router interface may have been created after the oid maps were refreshed at the start of this apply,
		// without its oid we can not wait for the asic to release it before the interface is recreated in another vrf
		if err := a.refreshRifOidMap(ctx); err != nil {
			return err
		}
		oid, known = a.rifOidMap[interfaceName]
	}

	a.log.Info("remove routing configuration for interface", "name", interfaceName)
	err = a.db.Config.DeleteInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not remove configuration for interface %s: %w", interfaceName, err)
	}

	if !known {
		if configured {
			a.log.Warn("router interface oid unknown, can not wait for the asic to release the interface", "name", interfaceName)
		}
		return nil
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
