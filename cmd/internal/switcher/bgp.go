package switcher

import (
	"text/template"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
)

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

func (r *renderer) move() (bool, error) {
	return move(r.tmp, r.dest)
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

func (a *BgpApplier) Apply(c *Conf) error {
	for _, r := range a.renderers {
		err := r.write(c)
		if err != nil {
			return err
		}
	}

	var anyMoved = false
	for _, r := range a.renderers {
		moved, err := r.move()
		if err != nil {
			return err
		}
		if moved {
			anyMoved = true
		}
	}

	if anyMoved {
		return dbus.Reload(bgpReloadService)
	}
	return nil
}
