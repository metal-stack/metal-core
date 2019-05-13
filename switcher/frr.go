package switcher

import (
	"fmt"
	"io"

	"github.com/coreos/go-systemd/dbus"
)

const FrrValidationService = "frr-validation@"

// FrrApplier is responsible for writing and
// applying the FRR configuration
type FrrApplier struct {
	Conf *Conf
}

// NewFrrApplier creates a new FrrApplier
func NewFrrApplier(c *Conf) Applier {
	return FrrApplier{Conf: c}
}

// Render renders the frr configuration to the given writer
func (a FrrApplier) Render(w io.Writer) error {
	return render(frrTPL, *a.Conf, w)
}

// Validate validates the frr configuration given
func (a FrrApplier) Validate(f string) error {
	dbc, err := dbus.New()
	defer dbc.Close()
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %v", err)
	}

	c := make(chan string)
	_, err = dbc.StartUnit(fmt.Sprintf("%s@'%s'.service", FrrValidationService, f), "replace", c)
	if err != nil {
		return err
	}
	job := <-c
	if job != "done" {
		return fmt.Errorf("frr-validation failed %s", job)
	}
	return nil
}

// Reload reloads the necessary services
// when the frr configuration was changed
func (a FrrApplier) Reload() error {
	dbc, err := dbus.New()
	defer dbc.Close()
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %v", err)
	}

	c := make(chan string)
	_, err = dbc.ReloadUnit("frr.service", "replace", c)
	if err != nil {
		return err
	}
	job := <-c
	if job != "done" {
		return fmt.Errorf("frr reload job failed %s", job)
	}
	return nil
}
