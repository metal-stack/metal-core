package switcher

const (
	frr                  = "/etc/frr/frr.conf"
	frrTpl               = "frr.tpl"
	frrReloadService     = "frr.service"
	frrValidationService = "frr-validation"
)

func newFrrRenderer(tplPath string) *templateRenderer {
	if tplPath != "" {
		return &templateRenderer{mustParseFile(tplPath)}
	}
	return &templateRenderer{mustParseFS(frrTpl)}
}

func newFrrApplier(tplPath string) *networkApplier {
	d := newDestConfig(frr, newFrrRenderer(tplPath))
	r := dbusReloader{frrReloadService}
	v := dbusTemplateValidator{frrValidationService}

	return &networkApplier{
		destConfigs: []*destConfig{d},
		reloader:    &r,
		validator:   &v,
	}
}
