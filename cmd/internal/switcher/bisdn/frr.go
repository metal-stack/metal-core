package bisdn

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
)

const (
	frrConfFile = "/etc/frr/frr.conf"
	// frrService is BISDN's FRR unit; its reload runs frrinit.sh reload,
	// which applies frr.conf with frr-reload.py.
	frrService = "frr.service"
	// frrValidationService is installed by metal-roles and runs
	// vtysh --dryrun --inputfile on the given file.
	frrValidationService = "frr-validation"
)

func NewFrrApplier(log *slog.Logger, tplPath string) *templates.Applier {
	return templates.NewApplier(&templates.Config{
		Dest:              frrConfFile,
		Reloader:          reloadFrr,
		Tpl:               templates.BISDNFrrTemplate(tplPath),
		ValidationService: frrValidationService,
		Log:               log,
	})
}

func reloadFrr(ctx context.Context, previousConf string) error {
	err := dbus.Reload(ctx, frrService)
	if err == nil {
		return nil
	}

	errs := []error{fmt.Errorf("reloading %s failed: %w", frrService, err)}

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
