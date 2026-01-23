package cumulus

import (
	"context"

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
	return templates.NewApplier(&templates.Config{
		Dest:              interfacesConfFile,
		Reloader:          reloadInterfaces,
		Tpl:               templates.InterfacesTemplate(tplPath),
		ValidationService: interfacesValidationService,
	})
}

func NewFrrApplier(tplPath string) *templates.Applier {
	return templates.NewApplier(&templates.Config{
		Dest:              frrConfFile,
		Reloader:          reloadFrr,
		Tpl:               templates.CumulusFrrTemplate(tplPath),
		ValidationService: frrValidationService,
	})
}

func reloadInterfaces(ctx context.Context, _ string) error {
	return dbus.Start(ctx, interfacesReloadService)
}

func reloadFrr(ctx context.Context, _ string) error {
	return dbus.Reload(ctx, frrReloadService)
}
