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

	routedIn := a.portState.RoutedIn(interfaceName, vrfName)
	if current == vrfName {
		// a previous addition may still be in progress
		return a.waitFor(ctx, routedIn...)
	} else if len(current) != 0 {
		return fmt.Errorf("interface %s already member of a different vrfName %v", interfaceName, current)
	}

	// Otherwise the interface would be moved into the vrf on top of the router interface of a previous vrf.
	ready := append(a.portState.NotRouted(interfaceName), a.noRoutesVia("", interfaceName))
	if err := a.waitFor(ctx, ready...); err != nil {
		return fmt.Errorf("interface %s cannot be added to vrf %s: %w", interfaceName, vrfName, err)
	}

	a.log.Info("add interface to vrfName ", "name", interfaceName, "vrf", vrfName)
	if err := a.db.Config.SetVrfMember(ctx, interfaceName, vrfName); err != nil {
		return err
	}

	if err := a.waitFor(ctx, routedIn...); err != nil {
		return fmt.Errorf("interface %s was not added to vrf %s: %w", interfaceName, vrfName, err)
	}
	return nil
}
