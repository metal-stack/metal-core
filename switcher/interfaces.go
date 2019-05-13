package switcher

import (
	"fmt"
	"io"

	"github.com/coreos/go-systemd/dbus"
)

// InterfacesApplier is responsible for writing and
// applying the network interfaces configuration
type InterfacesApplier struct {
	Conf *Conf
}

// NewInterfacesApplier creates a new InterfacesApplier
func NewInterfacesApplier(c *Conf) Applier {
	return InterfacesApplier{Conf: c}
}

// Render renders the network interfaces to the given writer
func (a InterfacesApplier) Render(w io.Writer) error {
	return render(interfacesTPL, *a.Conf, w)
}

// Reload reloads the necessary services
// when the network interfaces configuration was changed
func (a InterfacesApplier) Reload() error {
	dbc, err := dbus.New()
	defer dbc.Close()
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %v", err)
	}
	c := make(chan string)
	_, err = dbc.StartUnit("ifreload.service", "replace", c)
	if err != nil {
		return err
	}
	job := <-c
	if job != "done" {
		return fmt.Errorf("ifreload job failed %s", job)
	}
	return nil
}
