package switcher

const (
	bgpd    = "/etc/sonic/frr/bgpd.conf"
	bgpdTpl = "bgpd.tpl"

	staticd    = "/etc/sonic/frr/staticd.conf"
	staticdTpl = "staticd.tpl"

	zebra    = "/etc/sonic/frr/zebra.conf"
	zebraTpl = "zebra.tpl"

	bgpReloadService     = "bgp.service"
	bgpValidationService = "bgp-validation"
)

func newBgpdRenderer() *templateRenderer {
	return &templateRenderer{mustParseFS(bgpdTpl)}
}

func newStaticdRenderer() *templateRenderer {
	return &templateRenderer{mustParseFS(staticdTpl)}
}

func newZebraRenderer() *templateRenderer {
	return &templateRenderer{mustParseFS(zebraTpl)}
}

func newBgpApplier() *networkApplier {
	r := dbusReloader{bgpReloadService}
	v := dbusTemplateValidator{bgpValidationService}

	ds := []*destConfig{
		newDestConfig(bgpd, newBgpdRenderer()),
		newDestConfig(staticd, newStaticdRenderer()),
		newDestConfig(zebra, newZebraRenderer()),
	}

	return &networkApplier{
		destConfigs: ds,
		reloader:    &r,
		validator:   &v,
	}
}
