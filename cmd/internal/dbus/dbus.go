package dbus

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
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
