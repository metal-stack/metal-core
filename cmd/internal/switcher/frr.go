package switcher

const (
	frr                  = "/etc/frr/frr.conf"
	frrTmp               = "/etc/frr/frr.tmp"
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
	r := dbusReloader{frrReloadService}
	v := dbusTemplateValidator{frrValidationService}

	return &networkApplier{
		dest:      frr,
		reloader:  &r,
		renderer:  newFrrRenderer(tplPath),
		tmpFile:   frrTmp,
		validator: &v,
	}
}
