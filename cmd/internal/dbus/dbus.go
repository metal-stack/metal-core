package dbus

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
	godbus "github.com/godbus/dbus/v5"
)

const done = "done"

func Reload(ctx context.Context, unitName string) error {
	dbc, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %w", err)
	}
	defer dbc.Close()

	c := make(chan string)
	_, err = dbc.ReloadUnitContext(ctx, unitName, "replace", c)
	if err != nil {
		return err
	}

	select {
	case job := <-c:
		if job != done {
			return fmt.Errorf("reloading %s failed", unitName)
		}
	case <-ctx.Done():
		return fmt.Errorf("reloading %s failed: %w", unitName, ctx.Err())
	}

	return nil
}

func Start(ctx context.Context, unitName string) error {
	dbc, err := dbus.NewWithContext(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %w", err)
	}
	defer dbc.Close()

	c := make(chan string)

	_, err = dbc.StartUnitContext(ctx, unitName, "replace", c)
	if err != nil {
		return err
	}

	select {
	case job := <-c:
		if job != done {
			return fmt.Errorf("start of %s failed", unitName)
		}
	case <-ctx.Done():
		return fmt.Errorf("start of %s failed: %w", unitName, ctx.Err())
	}

	return nil
}

// ReloadNetworkd makes systemd-networkd reload its .netdev and .network files,
// like networkctl reload. It creates new netdevs and reconfigures the links
// whose files changed.
func ReloadNetworkd(ctx context.Context) error {
	conn, err := godbus.ConnectSystemBus(godbus.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	obj := conn.Object("org.freedesktop.network1", "/org/freedesktop/network1")
	if err := obj.CallWithContext(ctx, "org.freedesktop.network1.Manager.Reload", 0).Err; err != nil {
		return fmt.Errorf("reloading systemd-networkd failed: %w", err)
	}
	return nil
}
