package redis

import (
	"context"
	"fmt"
)

func (a *Applier) ensureLinkLocalOnlyIsEnabled(ctx context.Context, interfaceName string) error {
	enabled, err := a.db.Config.IsLinkLocalOnly(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve interface status for %s: %w", interfaceName, err)
	}
	if enabled {
		return nil
	}
	return a.db.Config.EnableLinkLocalOnly(ctx, interfaceName)
}
