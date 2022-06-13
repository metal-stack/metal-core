package dbus

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
)

const done = "done"

func Reload(unitName string) error {
	dbc, err := dbus.NewWithContext(context.Background())
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %w", err)
	}
	defer dbc.Close()

	c := make(chan string)
	_, err = dbc.ReloadUnitContext(context.Background(), unitName, "replace", c)

	if err != nil {
		return err
	}

	job := <-c
	if job != done {
		return fmt.Errorf("reloading failed %s", job)
	}

	return nil
}

func Start(unitName string) error {
	dbc, err := dbus.NewWithContext(context.Background())
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %w", err)
	}
	defer dbc.Close()

	c := make(chan string)

	_, err = dbc.StartUnitContext(context.Background(), unitName, "replace", c)
	if err != nil {
		return err
	}

	job := <-c
	if job != done {
		return fmt.Errorf("start failed %s", job)
	}

	return nil
}
