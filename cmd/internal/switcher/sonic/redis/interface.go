package redis

import (
	"context"
	"fmt"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

func (a *Applier) ensureNotRouted(ctx context.Context, interfaceName string) error {
	notRouted := a.portState.NotRouted(interfaceName)

	pending, err := db.Pending(ctx, notRouted...)
	if err != nil {
		return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
	}
	if len(pending) == 0 {
		return nil
	}

	a.log.Info("remove routing configuration for interface", "name", interfaceName)
	err = a.db.Config.DeleteInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not remove configuration for interface %s: %w", interfaceName, err)
	}

	if err := a.waitFor(ctx, notRouted...); err != nil {
		return fmt.Errorf("interface %s is still routed: %w", interfaceName, err)
	}
	return nil
}
