package sonic

import (
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
)

const (
	frr                  = "/etc/sonic/frr/cumulus_frr.conf"
	frrTmp               = "/etc/sonic/frr/frr.tmp"
	frrReloadService     = "frr-reload.service"
	frrValidationService = "bgp-validation"
)

func NewFrrApplier(tplPath string) *templates.Applier {
	return &templates.Applier{
		Dest:              frr,
		Reloader:          reloadFrr,
		Tmp:               frrTmp,
		Tpl:               templates.SonicFrrTemplate(tplPath),
		ValidationService: frrValidationService,
	}
}

func reloadFrr() error {
	return dbus.Start(frrReloadService)
}
