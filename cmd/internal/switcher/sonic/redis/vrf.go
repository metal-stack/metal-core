package redis

import (
	"context"
	"fmt"
)

// ensureInterfaceIsVrfMember puts the interface into the given VRF.
//
// If the interface is already a routed interface in a different VRF (or in the default VRF),
// its routing configuration is removed first and the router interface is recreated in the desired VRF.
func (a *Applier) ensureInterfaceIsVrfMember(ctx context.Context, interfaceName, vrfName string) error {
	current, err := a.db.Config.GetVrfMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vrfName membership for %s from redis: %w", interfaceName, err)
	}

	if current == vrfName {
		return nil
	}

	if current != "" {
		a.log.Info("move interface to a different vrf", "name", interfaceName, "from", current, "to", vrfName)
	}
	err = a.ensureNotRouted(ctx, interfaceName)
	if err != nil {
		return err
	}

	a.log.Info("add interface to vrfName ", "name", interfaceName, "vrf", vrfName)
	return a.db.Config.SetVrfMember(ctx, interfaceName, vrfName)
}

// ensureNotVrfMember removes the interface from any non-default VRF, keeping it a routed interface in the default VRF.
func (a *Applier) ensureNotVrfMember(ctx context.Context, interfaceName string) error {
	current, err := a.db.Config.GetVrfMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vrfName membership for %s from redis: %w", interfaceName, err)
	}
	if current == "" {
		return nil
	}

	a.log.Info("remove interface from vrf", "name", interfaceName, "vrf", current)
	return a.ensureNotRouted(ctx, interfaceName)
}
