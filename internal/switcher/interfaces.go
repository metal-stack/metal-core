package switcher

import (
	"io"
	"text/template"

	"github.com/metal-stack/metal-networker/pkg/net"
)

const (
	// Interfaces is the path to the network interfaces file
	Interfaces = "/etc/network/interfaces"
	// InterfacesTmp is the path to a temporary location of the interfaces file
	InterfacesTmp = "/etc/network/interfaces.tmp"
	// InterfacesReloadService is the systemd service to reload
	InterfacesReloadService = "ifreload.service"
	// InterfacesValidationService is the systemd unit that is used for validation
	InterfacesValidationService = "interfaces-validation"
)

// InterfacesApplier is responsible for writing and
// applying the network interfaces configuration
type InterfacesApplier struct {
	applier net.Applier
}

// NewInterfacesApplier creates a new InterfacesApplier
func NewInterfacesApplier(c *Conf) Applier {
	v := net.DBusTemplateValidator{TemplateName: InterfacesValidationService, InstanceName: InterfacesTmp}
	r := net.DBusStartReloader{ServiceFilename: InterfacesReloadService}
	a := net.NewNetworkApplier(c, v, r)
	return InterfacesApplier{a}
}

// Apply applies the configuration to the system
func (a InterfacesApplier) Apply() error {
	tpl := template.Must(template.New(interfacesTPL).Parse(interfacesTPL))
	return a.applier.Apply(*tpl, InterfacesTmp, Interfaces, true)
}

// Render renders the network interfaces to the given writer
func (a InterfacesApplier) Render(w io.Writer) error {
	tpl := template.Must(template.New(interfacesTPL).Parse(interfacesTPL))
	return a.applier.Render(w, *tpl)
}
