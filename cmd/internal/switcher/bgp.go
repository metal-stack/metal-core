package switcher

import (
	"text/template"
)

const (
	bgpd    = "/etc/sonic/frr/bgpd.conf"
	bgpdTpl = "bgpd.tpl"

	staticd    = "/etc/sonic/frr/staticd.conf"
	staticdTpl = "staticd.tpl"

	zebra    = "/etc/sonic/frr/zebra.conf"
	zebraTpl = "zebra.tpl"

	bgpValidationService = "bgp-validation"
)

type renderer struct {
	dest string
	tmp  string
	tpl  *template.Template
}

func newRenderer(dest string, tpl *template.Template) *renderer {
	return &renderer{
		dest: dest,
		tmp:  dest + ".tmp",
		tpl:  tpl,
	}
}

func (r *renderer) write(c *Conf) error {
	err := write(c, r.tpl, r.tmp)
	if err != nil {
		return err
	}
	return validate(bgpValidationService, r.tmp)
}

type BgpApplier struct {
	renderers []*renderer
}

func newBgpApplier() *BgpApplier {
	return &BgpApplier{
		renderers: []*renderer{
			newRenderer(bgpd, mustParseFS(bgpdTpl)),
			newRenderer(staticd, mustParseFS(staticdTpl)),
			newRenderer(zebra, mustParseFS(zebraTpl)),
		},
	}
}

func (a *BgpApplier) Apply(c *Conf) (applied bool, err error) {
	for _, r := range a.renderers {
		err := r.write(c)
		if err != nil {
			return false, err
		}
	}

	for _, r := range a.renderers {
		a, err := move(r.tmp, r.dest)
		if err != nil {
			return false, err
		}

		if a {
			applied = true
		}
	}

	return applied, nil
}
