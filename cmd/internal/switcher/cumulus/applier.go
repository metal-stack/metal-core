package cumulus

import (
	"context"
	"log/slog"

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

func NewInterfacesApplier(log *slog.Logger, tplPath string) *templates.Applier {
	return templates.NewApplier(&templates.Config{
		Dest:              interfacesConfFile,
		Reloader:          reloadInterfaces,
		Tpl:               templates.InterfacesTemplate(tplPath),
		ValidationService: interfacesValidationService,
		Log:               log,
	})
}

func NewFrrApplier(log *slog.Logger, tplPath string) *templates.Applier {
	return templates.NewApplier(&templates.Config{
		Dest:              frrConfFile,
		Reloader:          reloadFrr,
		Tpl:               templates.CumulusFrrTemplate(tplPath),
		ValidationService: frrValidationService,
		Log:               log,
	})
}

func reloadInterfaces(ctx context.Context, _ string) error {
	return dbus.Start(ctx, interfacesReloadService)
}

func reloadFrr(ctx context.Context, _ string) error {
	return dbus.Reload(ctx, frrReloadService)
}
