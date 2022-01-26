package switcher

import (
	"fmt"
	"io"
	"path"
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
	tplFile string
	applier net.Applier
}

// NewInterfacesApplier creates a new InterfacesApplier
func NewInterfacesApplier(c *Conf) Applier {
	v := net.DBusTemplateValidator{TemplateName: InterfacesValidationService, InstanceName: InterfacesTmp}
	r := net.DBusStartReloader{ServiceFilename: InterfacesReloadService}
	a := net.NewNetworkApplier(c, v, r)
	return InterfacesApplier{
		applier: a,
		tplFile: c.InterfacesTplFile,
	}
}

// Apply applies the configuration to the system
func (a InterfacesApplier) Apply() error {
	tpl := a.getTpl()
	ok, err := a.applier.Apply(*tpl, InterfacesTmp, Interfaces, true)
	if !ok {
		return fmt.Errorf("interface changes have not been applied %w", err)
	}
	return err
}

// Render renders the network interfaces to the given writer
func (a InterfacesApplier) Render(w io.Writer) error {
	tpl := a.getTpl()
	return a.applier.Render(w, *tpl)
}

func (a InterfacesApplier) getTpl() *template.Template {
	if a.tplFile != "" {
		return template.Must(template.New(path.Base(a.tplFile)).ParseFiles(a.tplFile))
	}
	return template.Must(template.New("interfaces.tpl").Parse(interfacesTPL))
}
