package templates

import (
	"text/template"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type FrrApplier struct {
	dest              string
	tmp               string
	validationService string
	reloadService     string
	tpl               *template.Template
}

func NewFrrApplier(dest, tmp, validationService, reloadService, tplPath string, embedFS bool) *FrrApplier {
	if !embedFS {
		return &FrrApplier{
			dest:              dest,
			tmp:               tmp,
			validationService: validationService,
			reloadService:     reloadService,
			tpl:               mustParseFile(tplPath),
		}
	}

	return &FrrApplier{
		dest:              dest,
		tmp:               tmp,
		validationService: validationService,
		reloadService:     reloadService,
		tpl:               mustParseFS(tplPath),
	}
}

func (a *FrrApplier) Apply(c *types.Conf) (applied bool, err error) {
	err = write(c, a.tpl, a.tmp)
	if err != nil {
		return false, err
	}

	err = validate(a.validationService, a.tmp)
	if err != nil {
		return false, err
	}

	moved, err := move(a.tmp, a.dest)
	if err == nil && moved && a.reloadService != "" {
		return true, dbus.Reload(a.reloadService)
	}
	return moved, err
}
