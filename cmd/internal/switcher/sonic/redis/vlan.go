package redis

import (
	"context"
	"fmt"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

func (a *Applier) ensureNotBridged(ctx context.Context, interfaceName string) error {
	notBridged := a.portState.NotBridged(interfaceName)

	pending, err := db.Pending(ctx, notBridged...)
	if err != nil {
		return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
	}
	if len(pending) == 0 {
		return nil
	}

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

	if err := a.waitFor(ctx, notBridged...); err != nil {
		return fmt.Errorf("interface %s is still bridged: %w", interfaceName, err)
	}
	return nil
}

func (a *Applier) ensureInterfaceIsVlanMember(ctx context.Context, interfaceName, vlan string) error {
	current, err := a.db.Config.GetVlanMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan current for %s from redis: %w", interfaceName, err)
	}

	if len(current) == 1 && current[0] == vlan {
		return nil
	} else if len(current) != 0 {
		return fmt.Errorf("interface %s already member of a different vlan %v", interfaceName, current)
	}

	a.log.Info("add interface to vlan", "name", interfaceName, "vlan", vlan)
	return a.db.Config.SetVlanMember(ctx, interfaceName, vlan)
}
