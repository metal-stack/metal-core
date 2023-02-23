package templates

import (
	"text/template"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const (
	frr                  = "/etc/frr/frr.conf"
	frrTmp               = "/etc/frr/frr.tmp"
	frrTpl               = "frr.tpl"
	frrReloadService     = "frr.service"
	frrValidationService = "frr-validation"
)

type FrrApplier struct {
	tpl *template.Template
}

func NewFrrApplier(tplPath string) *FrrApplier {
	if tplPath != "" {
		return &FrrApplier{mustParseFile(tplPath)}
	}
	return &FrrApplier{mustParseFS(frrTpl)}
}

func (a *FrrApplier) Apply(c *types.Conf) error {
	err := write(c, a.tpl, frrTmp)
	if err != nil {
		return err
	}

	err = validate(frrValidationService, frrTmp)
	if err != nil {
		return err
	}

	moved, err := move(frrTmp, frr)
	if err == nil && moved {
		return dbus.Reload(frrReloadService)
	}
	return err
}
