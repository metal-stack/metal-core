package redis

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
)

// ensureNotBridged removes the interface from all VLANs and waits until the bridge port is gone from the ASIC.
func (a *Applier) ensureNotBridged(ctx context.Context, interfaceName string) error {
	vlans, err := a.db.Config.GetVlanMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vlan membership for %s from Redis: %w", interfaceName, err)
	}

	var bridged bool
	oid, known := a.bridgePortOidMap[interfaceName]
	if known {
		bridged, err = a.db.Asic.ExistBridgePort(ctx, oid)
		if err != nil {
			return fmt.Errorf("could not retrieve state data for interface %s: %w", interfaceName, err)
		}
	}

	if len(vlans) == 0 && !bridged {
		return nil
	}

	for _, vlan := range vlans {
		a.log.Info("remove interface from vlan", "name", interfaceName, "vlan", vlan)
		err := a.db.Config.DeleteVlanMember(ctx, interfaceName, vlan)
		if err != nil {
			return fmt.Errorf("could not remove interface %s from vlan %s", interfaceName, vlan)
		}
	}

	if !known {
		return nil
	}

	return retry.Do(
		func() error {
			bridged, err := a.db.Asic.ExistBridgePort(ctx, oid)
			if err != nil {
				return err
			}
			if bridged {
				return fmt.Errorf("interface %s still bridged", interfaceName)
			}
			return nil
		},
	)
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
