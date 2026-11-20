package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const restoreTimeout = 5 * time.Second

func (a *Applier) ensureNotRouted(ctx context.Context, interfaceName string) error {
	configured, err := a.db.Config.ExistInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve configuration for interface %s: %w", interfaceName, err)
	}
	if configured {
		if err := a.removeInterfaceConfiguration(ctx, interfaceName); err != nil {
			return err
		}
	}

	// also without a configuration, as a previous removal may still be in progress
	if err := a.waitFor(ctx, a.portState.NotRouted(interfaceName)...); err != nil {
		return fmt.Errorf("interface %s is still routed: %w", interfaceName, err)
	}
	return nil
}

// removeInterfaceConfiguration takes the interface out of its vrf once no routes use it anymore.
//
// Routes via the interface keep its router interface referenced in orchagent, so it cannot be removed before.
// Taking the port down drops the BGP session while the interface is still in its vrf, so the withdrawn
// routes reach the APPL_DB. Routes withdrawn after the interface left its vrf may never be removed from it.
func (a *Applier) removeInterfaceConfiguration(ctx context.Context, interfaceName string) (err error) {
	vrf, err := a.db.Config.GetVrfMembership(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not retrieve vrf membership for %s: %w", interfaceName, err)
	}

	restore, err := a.takeDown(ctx, interfaceName)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, restore())
	}()

	a.log.Info("wait until no routes use interface", "name", interfaceName, "vrf", vrf)
	if err := a.waitFor(ctx, a.noRoutesVia(vrf, interfaceName)); err != nil {
		return fmt.Errorf("not removing routing configuration for interface %s: %w", interfaceName, err)
	}

	a.log.Info("remove routing configuration for interface", "name", interfaceName)
	err = a.db.Config.DeleteInterfaceConfiguration(ctx, interfaceName)
	if err != nil {
		return fmt.Errorf("could not remove configuration for interface %s: %w", interfaceName, err)
	}
	return nil
}

// takeDown sets the port admin down and returns how to bring it up again.
func (a *Applier) takeDown(ctx context.Context, interfaceName string) (restore func() error, err error) {
	p, err := a.db.Config.GetPort(ctx, interfaceName)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve port info for %s from redis: %w", interfaceName, err)
	}
	if p == nil {
		return nil, fmt.Errorf("port %s does not exist in CONFIG_DB", interfaceName)
	}
	if p.AdminStatus == string(types.PortStatusDown) {
		return func() error { return nil }, nil
	}

	a.log.Info("take port down", "name", interfaceName)
	if err := a.db.Config.SetAdminStatus(ctx, interfaceName, types.PortStatusDown); err != nil {
		return nil, fmt.Errorf("could not take port %s down: %w", interfaceName, err)
	}

	return func() error {
		// the port must not stay down because the reconfiguration timed out
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restoreTimeout)
		defer cancel()

		a.log.Info("bring port up again", "name", interfaceName)
		if err := a.db.Config.SetAdminStatus(ctx, interfaceName, types.PortStatusUp); err != nil {
			return fmt.Errorf("could not bring port %s up again: %w", interfaceName, err)
		}
		return nil
	}, nil
}

// noRoutesVia holds while no route in the APPL_DB uses the interface as next hop. An empty vrf checks all vrfs.
func (a *Applier) noRoutesVia(vrf, interfaceName string) db.Condition {
	desc := fmt.Sprintf("APPL_DB no routes via %s", interfaceName)
	if vrf != "" {
		desc += " in " + vrf
	}
	return db.NewCondition(desc, func(ctx context.Context) (bool, string, error) {
		routes, err := a.db.Appl.GetRoutesVia(ctx, vrf, interfaceName)
		return len(routes) == 0, strings.Join(routes, ", "), err
	})
}
