package switcher

import (
	"text/template"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
)

const (
	interfaces                  = "/etc/network/interfaces"
	interfacesTmp               = "/etc/network/interfaces.tmp"
	interfacesTpl               = "interfaces.tpl"
	interfacesReloadService     = "ifreload.service"
	interfacesValidationService = "interfaces-validation"
)

type InterfacesApplier struct {
	tpl *template.Template
}

func NewInterfacesApplier(tplPath string) *InterfacesApplier {
	if tplPath != "" {
		return &InterfacesApplier{mustParseFile(tplPath)}
	}
	return &InterfacesApplier{mustParseFS(frrTpl)}
}

func (a *InterfacesApplier) Apply(c *Conf) error {
	err := write(c, a.tpl, interfacesTmp)
	if err != nil {
		return err
	}

	err = validate(interfacesValidationService, interfacesTmp)
	if err != nil {
		return err
	}

	moved, err := move(interfacesTmp, interfaces)
	if err == nil && moved {
		return dbus.Start(interfacesReloadService)
	}
	return err
}
