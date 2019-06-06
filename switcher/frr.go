package switcher

import (
	"io"
	"text/template"

	"git.f-i-ts.de/cloud-native/metallib/network"
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
	applier network.Applier
}

// NewFrrApplier creates a new FrrApplier
func NewFrrApplier(c *Conf) Applier {
	v := network.DBusTemplateValidator{TemplateName: FrrValidationService, InstanceName: FrrTmp}
	r := network.DBusReloader{ServiceFilename: FrrReloadService}
	a := network.NewNetworkApplier(c, v, r)
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
