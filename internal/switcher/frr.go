package switcher

import (
	"io"
	"text/template"

	"github.com/metal-stack/metal-networker/pkg/net"
)

const (
	// Frr is the path to the frr configuration file
	Frr = "/etc/frr/frr.conf"
	// FrrTmp is the path to the tempoary location of the frr configuration file
	FrrTmp = "/etc/frr/frr.tmp"
	// FrrReloadService is the systemd service to reload
	FrrReloadService = "frr.service"
	// FrrValidationService is the systemd unit that is used for validation
	FrrValidationService = "frr-validation"
)

// FrrApplier is responsible for writing and
// applying the FRR configuration
type FrrApplier struct {
	applier net.Applier
}

// NewFrrApplier creates a new FrrApplier
func NewFrrApplier(c *Conf) Applier {
	v := net.DBusTemplateValidator{TemplateName: FrrValidationService, InstanceName: FrrTmp}
	r := net.DBusReloader{ServiceFilename: FrrReloadService}
	a := net.NewNetworkApplier(c, v, r)
	return FrrApplier{a}
}

// Apply applies the configuration to the system
func (a FrrApplier) Apply() error {
	tpl := template.Must(template.New(frrTPL).Parse(frrTPL))
	return a.applier.Apply(*tpl, FrrTmp, Frr, true)
}

// Render renders the frr configuration to the given writer
func (a FrrApplier) Render(w io.Writer) error {
	tpl := template.Must(template.New(frrTPL).Parse(frrTPL))
	return a.applier.Render(w, *tpl)
}
