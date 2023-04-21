package sonic

import (
	"fmt"

	"github.com/coreos/go-systemd/v22/unit"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
)

func NewRemoveNeighborsApplier() *templates.Applier {
	dest := "/etc/sonic/frr/unprovisioned"
	reloader := fmt.Sprintf("remove-neighbors@%s.service", unit.UnitNamePathEscape(dest))

	return &templates.Applier{
		Dest: "/etc/sonic/frr/unprovisioned",
		Reloader: func() error {
			return dbus.Start(reloader)
		},
		Tmp: dest + ".tmp",
		Tpl: templates.UnprovisionedTemplate(),
	}
}
