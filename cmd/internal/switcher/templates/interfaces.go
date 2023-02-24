package templates

import (
	"text/template"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
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
	return &InterfacesApplier{mustParseFS(interfacesTpl)}
}

func (a *InterfacesApplier) Apply(c *types.Conf) (applied bool, err error) {
	err = write(c, a.tpl, interfacesTmp)
	if err != nil {
		return false, err
	}

	err = validate(interfacesValidationService, interfacesTmp)
	if err != nil {
		return false, err
	}

	moved, err := move(interfacesTmp, interfaces)
	if err != nil {
		return false, err
	}
	if moved {
		return moved, dbus.Start(interfacesReloadService)
	}
	return false, nil
}
