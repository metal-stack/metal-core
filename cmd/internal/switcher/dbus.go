package switcher

import (
	"fmt"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
)

type dbusReloader struct {
	unitName string
}

func (r *dbusReloader) Reload() error {
	return dbus.Reload(r.unitName)
}

type dbusStartReloader struct {
	unitName string
}

func (r *dbusStartReloader) Reload() error {
	return dbus.Start(r.unitName)
}

type dbusTemplateValidator struct {
	templateName string
}

func (v *dbusTemplateValidator) Validate(path string) error {
	u := fmt.Sprintf("%s@%s.service", v.templateName, unit.UnitNamePathEscape(path))
	if err := dbus.Start(u); err != nil {
		return fmt.Errorf("validation failed %w", err)
	}
	return nil
}
