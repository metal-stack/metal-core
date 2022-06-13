package switcher

const (
	interfaces                  = "/etc/network/interfaces"
	interfacesTmp               = "/etc/network/interfaces.tmp"
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
	r := dbusStartReloader{interfacesReloadService}
	v := dbusTemplateValidator{interfacesValidationService}

	return &networkApplier{
		dest:      interfaces,
		reloader:  &r,
		renderer:  newInterfacesRenderer(tplPath),
		tmpFile:   interfacesTmp,
		validator: &v,
	}
}
