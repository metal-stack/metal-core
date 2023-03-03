package cumulus

import (
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
)

const (
	frr                  = "/etc/frr/frr.conf"
	frrTmp               = "/etc/frr/frr.tmp"
	frrReloadService     = "frr.service"
	frrValidationService = "frr-validation"

	interfaces                  = "/etc/network/interfaces"
	interfacesTmp               = "/etc/network/interfaces.tmp"
	interfacesReloadService     = "ifreload.service"
	interfacesValidationService = "interfaces-validation"
)

func NewInterfacesApplier(tplPath string) *templates.Applier {
	return &templates.Applier{
		Dest:              interfaces,
		Reloader:          reloadInterfaces,
		Tmp:               interfacesTmp,
		Tpl:               templates.InterfacesTemplate(tplPath),
		ValidationService: interfacesValidationService,
	}
}

func NewFrrApplier(tplPath string) *templates.Applier {
	return &templates.Applier{
		Dest:              frr,
		Reloader:          reloadFrr,
		Tmp:               frrTmp,
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
