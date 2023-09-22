package sonic

import (
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
)

const (
	frrConfFile          = "/etc/sonic/frr/frr.conf"
	frrReloadService     = "frr-reload.service"
	frrValidationService = "bgp-validation"
)

func NewFrrApplier(tplPath string) *templates.Applier {
	return &templates.Applier{
		Dest:              frrConfFile,
		Reloader:          reloadFrr,
		Tpl:               templates.SonicFrrTemplate(tplPath),
		ValidationService: frrValidationService,
	}
}

func reloadFrr() error {
	return dbus.Start(frrReloadService)
}
