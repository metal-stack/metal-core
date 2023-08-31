package redis

import (
	"context"
	"fmt"
)

func (a *Applier) ensureInterfaceIsVrfMember(ctx context.Context, interfaceName, vrfName string) error {
	current, err := a.db.Config.GetVrfMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vrfName membership for %s from redis: %w", interfaceName, err)
	}

	if current == vrfName {
		return nil
	} else if len(current) != 0 {
		return fmt.Errorf("interface %s already member of a different vrfName %v", interfaceName, current)
	}

	a.log.Info("add interface to vrfName ", "name", interfaceName, "vrf", vrfName)
	return a.db.Config.SetVrfMember(ctx, interfaceName, vrfName)
}
