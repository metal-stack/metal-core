package switcher

import (
	"git.f-i-ts.de/cloud-native/metallib/network"
	"io"
	"text/template"
)

const (
	IfacesTmp = "/etc/network/interfaces.tmp"
	InterfacesValidationService = "interfaces-validation"
)


// InterfacesApplier is responsible for writing and
// applying the network interfaces configuration
type InterfacesApplier struct {
	applier network.NetworkApplier
}

// NewInterfacesApplier creates a new InterfacesApplier
func NewInterfacesApplier(c *Conf) Applier {
	a := network.NewNetworkApplier(c)
	return InterfacesApplier{a}
}

// Render renders the network interfaces to the given writer
func (a InterfacesApplier) Render(w io.Writer) error {
	tpl := template.Must(template.New(interfacesTPL).Parse(interfacesTPL))
	return a.applier.Render(w, *tpl)
}

func (a InterfacesApplier) Validate() error {
	v := network.DBusTemplateValidator{InterfacesValidationService, IfacesTmp}
	return a.applier.Validate(v)
}

// Reload reloads the necessary services
// when the network interfaces configuration was changed
func (a InterfacesApplier) Reload() error {
	r := network.DBusStartReloader{"ifreload.service"}
	return a.applier.Reload(r)
}
