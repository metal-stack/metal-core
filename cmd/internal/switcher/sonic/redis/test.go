package redis

import (
	"context"
	"fmt"
)

func (e *testASIC) EnsureNotRouted(ctx context.Context, interfaceName string) error {
	err := e.db.Config.DeleteInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not remove configuration for interface %s: %w", interfaceName, err)
	}

	return nil
}

func (e *testASIC) EnsureNotBridged(ctx context.Context, interfaceName string) error {
	vlans, err := e.db.Config.GetVlanMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan membership for %s from Redis: %w", interfaceName, err)
	}

	for _, vlan := range vlans {
		err := e.db.Config.DeleteVlanMember(ctx, interfaceName, vlan)
		if err != nil {
			return fmt.Errorf("could not remove interface %s from vlan %s", interfaceName, vlan)
		}
	}

	return nil
}
