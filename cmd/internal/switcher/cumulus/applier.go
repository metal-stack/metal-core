package cumulus

import (
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
)

const (
	frrConfFile          = "/etc/frr/frr.conf"
	frrReloadService     = "frr.service"
	frrValidationService = "frr-validation"

	interfacesConfFile          = "/etc/network/interfaces"
	interfacesReloadService     = "ifreload.service"
	interfacesValidationService = "interfaces-validation"
)

func NewInterfacesApplier(tplPath string) *templates.Applier {
	return &templates.Applier{
		Dest:              interfacesConfFile,
		Reloader:          reloadInterfaces,
		Tpl:               templates.InterfacesTemplate(tplPath),
		ValidationService: interfacesValidationService,
	}
}

func NewFrrApplier(tplPath string) *templates.Applier {
	return &templates.Applier{
		Dest:              frrConfFile,
		Reloader:          reloadFrr,
		Tpl:               templates.CumulusFrrTemplate(tplPath),
		ValidationService: frrValidationService,
	}
}

func reloadInterfaces() error {
	return dbus.Start(interfacesReloadService)
}

func reloadFrr() error {
	return dbus.Reload(frrReloadService)
}
