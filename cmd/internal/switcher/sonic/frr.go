package sonic

import (
	"errors"
	"fmt"
	"os"

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

func reloadFrr(previousConf string) error {
	err := dbus.Start(frrReloadService)
	if err == nil {
		return nil
	}

	errs := []error{fmt.Errorf("reloading %s failed: %w", frrReloadService, err)}

	if previousConf != "" {
		err = os.Rename(previousConf, frrConfFile)
		if err == nil {
			return errors.Join(errs...)
		}
		errs = append(errs, fmt.Errorf("could not restore %s from %s: %w", frrConfFile, previousConf, err))
	}

	err = os.Remove(frrConfFile)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to remove %s: %w", frrConfFile, err))
	}
	return errors.Join(errs...)
}
