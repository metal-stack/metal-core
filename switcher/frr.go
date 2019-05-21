package switcher

import (
	"git.f-i-ts.de/cloud-native/metallib/network"
	"io"
	"text/template"
)

const (
	FrrTmp = "/etc/frr/frr.tmp"
	FrrValidationService = "frr-validation"
)
// FrrApplier is responsible for writing and
// applying the FRR configuration
type FrrApplier struct {
	applier network.NetworkApplier
}

// NewFrrApplier creates a new FrrApplier
func NewFrrApplier(c *Conf) Applier {
	a := network.NewNetworkApplier(c)
	return FrrApplier{a}
}

// Render renders the frr configuration to the given writer
func (a FrrApplier) Render(w io.Writer) error {
	tpl := template.Must(template.New(frrTPL).Parse(frrTPL))
	return a.applier.Render(w, *tpl)
}

func (a FrrApplier) Validate() error {
	v := network.DBusTemplateValidator{FrrValidationService, FrrTmp}
	return a.applier.Validate(v)
}

// Reload reloads the necessary services
// when the frr configuration was changed
func (a FrrApplier) Reload() error {
	r := network.DBusReloader{"frr.service"}
	return a.applier.Reload(r)
}
