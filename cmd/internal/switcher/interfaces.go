package switcher

const (
	interfaces                  = "/etc/network/interfaces"
	interfacesTpl               = "interfaces.tpl"
	interfacesReloadService     = "ifreload.service"
	interfacesValidationService = "interfaces-validation"
)

func newInterfacesRenderer(tplPath string) *templateRenderer {
	if tplPath != "" {
		return &templateRenderer{mustParseFile(tplPath)}
	}
	return &templateRenderer{mustParseFS(interfacesTpl)}
}

func newInterfacesApplier(tplPath string) *networkApplier {
	d := newDestConfig(interfaces, newInterfacesRenderer(tplPath))
	r := dbusStartReloader{interfacesReloadService}
	v := dbusTemplateValidator{interfacesValidationService}

	return &networkApplier{
		destConfigs: []*destConfig{d},
		reloader:    &r,
		validator:   &v,
	}
}
