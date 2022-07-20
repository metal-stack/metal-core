package switcher

import (
	"io"
	"path"
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
	tplFile string
	applier net.Applier
}

// NewFrrApplier creates a new FrrApplier
func NewFrrApplier(c *Conf, tplFile string) Applier {
	v := net.DBusTemplateValidator{TemplateName: FrrValidationService, InstanceName: FrrTmp}
	r := net.DBusReloader{ServiceFilename: FrrReloadService}
	a := net.NewNetworkApplier(c, v, r)
	return FrrApplier{
		applier: a,
		tplFile: tplFile,
	}
}

// Apply applies the configuration to the system
func (a FrrApplier) Apply() error {
	tpl := a.getTpl()
	_, err := a.applier.Apply(*tpl, FrrTmp, Frr, true)
	return err
}

// Render renders the frr configuration to the given writer
func (a FrrApplier) Render(w io.Writer) error {
	tpl := a.getTpl()
	return a.applier.Render(w, *tpl)
}

func (a FrrApplier) getTpl() *template.Template {
	if a.tplFile != "" {
		return template.Must(template.New(path.Base(a.tplFile)).ParseFiles(a.tplFile))
	}
	return template.Must(template.New("frr.tpl").Parse(frrTPL))
}
