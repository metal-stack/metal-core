package redis

import (
	"context"
	"fmt"
)

func (a *Applier) ensureNotBridged(ctx context.Context, interfaceName string) error {
	vlans, err := a.db.Config.GetVlanMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan membership for %s from Redis: %w", interfaceName, err)
	}

	for _, vlan := range vlans {
		a.log.Info("remove interface from vlan", "name", interfaceName, "vlan", vlan)
		err := a.db.Config.DeleteVlanMember(ctx, interfaceName, vlan)
		if err != nil {
			return fmt.Errorf("could not remove interface %s from vlan %s", interfaceName, vlan)
		}
	}

	// also without a membership, as a previous removal may still be in progress
	if err := a.waitFor(ctx, a.portState.NotBridged(interfaceName)...); err != nil {
		return fmt.Errorf("interface %s is still bridged: %w", interfaceName, err)
	}
	return nil
}

func (a *Applier) ensureInterfaceIsVlanMember(ctx context.Context, interfaceName, vlan string) error {
	current, err := a.db.Config.GetVlanMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan current for %s from redis: %w", interfaceName, err)
	}

	bridgedIn := a.portState.BridgedIn(interfaceName, vlan)
	if len(current) == 1 && current[0] == vlan {
		// a previous addition may still be in progress
		return a.waitFor(ctx, bridgedIn...)
	} else if len(current) != 0 {
		return fmt.Errorf("interface %s already member of a different vlan %v", interfaceName, current)
	}

	a.log.Info("add interface to vlan", "name", interfaceName, "vlan", vlan)
	if err := a.db.Config.SetVlanMember(ctx, interfaceName, vlan); err != nil {
		return err
	}

	if err := a.waitFor(ctx, bridgedIn...); err != nil {
		return fmt.Errorf("interface %s was not added to vlan %s: %w", interfaceName, vlan, err)
	}
	return nil
}
