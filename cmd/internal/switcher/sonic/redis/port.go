package redis

import (
	"context"
	"fmt"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func (a *Applier) configurePort(ctx context.Context, port types.Port) error {
	p, err := a.db.Config.GetPort(ctx, port.Name)
	if err != nil {
		return fmt.Errorf("could not retrieve port info for %s from redis: %w", port.Name, err)
	}
	if p == nil {
		return fmt.Errorf("port %s does not exist in CONFIG_DB", port.Name)
	}

	if port.VRF != nil && port.VLAN != nil {
		return fmt.Errorf("cannot add port %s to a vrf and a vlan at the same time", port.Name)
	}

	if p.Mtu != fmt.Sprintf("%d", port.MTU) {
		a.log.Debug("set port mtu to", "port", port.Name, "mtu", port.MTU)
		err = a.db.Config.SetPortMtu(ctx, port.Name, fmt.Sprintf("%d", port.MTU))
		if err != nil {
			return err
		}
	}

	if p.AdminStatus != string(port.AdminStatus) && port.AdminStatus != "" {
		a.log.Debug("set admin status to", "port", port.Name, "admin_status", port.AdminStatus)
		return a.db.Config.SetAdminStatus(ctx, port.Name, port.AdminStatus)
	}

	if port.VLAN != nil {
		err := a.configureBridgedPort(ctx, port)
		if err != nil {
			return fmt.Errorf("failed to configure bridged port %s: %w", port.Name, err)
		}
		return nil
	}

	err = a.configureRoutedPort(ctx, port)
	if err != nil {
		return fmt.Errorf("failed to configure routed port %s: %w", port.Name, err)
	}

	if port.VRF != nil {
		err = a.ensureInterfaceIsVrfMember(ctx, port.Name, pointer.SafeDeref(port.VRF))
		if err != nil {
			return fmt.Errorf("failed to ensure port %s is member of vrf %s: %w", port.Name, pointer.SafeDeref(port.VRF), err)
		}
	}

	return nil
}

func (a *Applier) configureRoutedPort(ctx context.Context, port types.Port) error {
	err := a.asic.EnsureNotBridged(ctx, port.Name)
	if err != nil {
		return fmt.Errorf("failed to ensure port %s is not bridged: %w", port.Name, err)
	}

	err = a.ensureLinkLocalOnlyIsEnabled(ctx, port.Name)
	if err != nil {
		return fmt.Errorf("failed to ensure link local only is enabled for port %s: %w", port.Name, err)
	}

	return nil
}

func (a *Applier) configureBridgedPort(ctx context.Context, port types.Port) error {
	err := a.asic.EnsureNotRouted(ctx, port.Name)
	if err != nil {
		return fmt.Errorf("failed to ensure port %s is not routed: %w", port.Name, err)
	}

	err = a.ensureInterfaceIsVlanMember(ctx, port.Name, pointer.SafeDeref(port.VLAN))
	if err != nil {
		return fmt.Errorf("failed to ensure port %s is member of vlan %s: %w", port.Name, pointer.SafeDeref(port.VLAN), err)
	}

	return nil
}
