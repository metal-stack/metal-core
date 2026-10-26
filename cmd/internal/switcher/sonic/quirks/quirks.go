package quirks

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/frrfpm"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/frrkeepinterfaces"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/frrnexthopgroups"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/lldp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/portstate"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/vrfentries"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

var QuirkProfiles = map[string]func() SonicQuirks{
	"community_old": func() SonicQuirks {
		return SonicQuirks{
			VrfEntries:              vrfentries.Community{},
			PortState:               portstate.AsicDb{},
			FrrUseDplaneFpmSonic:    frrfpm.Pre202411{},
			FrrDisableNextHopGroups: frrnexthopgroups.Pre202411{},
			FrrKeepInterfaces:       frrkeepinterfaces.Compat{},
			LLDPConfiguration:       lldp.Community{},
		}
	},
	"community_2025": func() SonicQuirks {
		return SonicQuirks{
			VrfEntries:              vrfentries.Community{},
			PortState:               portstate.AsicDb{},
			FrrUseDplaneFpmSonic:    frrfpm.Post202411{},
			FrrDisableNextHopGroups: frrnexthopgroups.Post202411{},
			FrrKeepInterfaces:       frrkeepinterfaces.Compat{},
			LLDPConfiguration:       lldp.Community{},
		}
	},
	"community_vpp": func() SonicQuirks {
		return SonicQuirks{
			VrfEntries:              vrfentries.Community{},
			PortState:               portstate.AsicDb{},
			FrrUseDplaneFpmSonic:    frrfpm.Post202411{},
			FrrDisableNextHopGroups: frrnexthopgroups.Post202411{},
			FrrKeepInterfaces:       frrkeepinterfaces.Compat{},
			LLDPConfiguration:       lldp.Community{},
		}
	},
	"edgecore": func() SonicQuirks {
		return SonicQuirks{
			VrfEntries:              vrfentries.Community{},
			PortState:               portstate.AsicDb{},
			FrrUseDplaneFpmSonic:    frrfpm.Pre202411{},
			FrrDisableNextHopGroups: frrnexthopgroups.Pre202411{},
			FrrKeepInterfaces:       frrkeepinterfaces.Compat{},
			LLDPConfiguration:       lldp.Community{},
		}
	},
	"broadcom_enterprise_2022": func() SonicQuirks {
		return SonicQuirks{
			VrfEntries:              vrfentries.Community{},
			PortState:               portstate.AsicDb{},
			FrrUseDplaneFpmSonic:    frrfpm.Pre202411{},
			FrrDisableNextHopGroups: frrnexthopgroups.Pre202411{},
			FrrKeepInterfaces:       frrkeepinterfaces.Compat{},
			LLDPConfiguration:       lldp.Community{},
		}
	},
	"asternos": func() SonicQuirks {
		return SonicQuirks{
			VrfEntries:              vrfentries.L3Only{},
			PortState:               portstate.StateDb{},
			FrrUseDplaneFpmSonic:    frrfpm.Post202411{},
			FrrDisableNextHopGroups: frrnexthopgroups.Post202411{},
			FrrKeepInterfaces:       frrkeepinterfaces.Modern{},
			LLDPConfiguration:       lldp.AsterNOS{},
		}
	},
}

// Runtime Quirks Configuration to allow for overriding SONiC distribution or version specific behavior later
// Check the Quirks directly for more information about what they allow you to configure
type SonicQuirks struct {
	VrfEntries              vrfentries.VrfEntries
	PortState               portstate.PortState
	FrrUseDplaneFpmSonic    frrfpm.FrrUseDplaneFpmSonic
	FrrDisableNextHopGroups frrnexthopgroups.FrrDisableNextHopGroups
	FrrKeepInterfaces       frrkeepinterfaces.FrrKeepInterfaces
	LLDPConfiguration       lldp.LLDPConfiguration
}

// LogValue lists the behavior selected for each quirk, e.g. VrfEntries=vrfentries.L3Only
func (q SonicQuirks) LogValue() slog.Value {
	v := reflect.ValueOf(q)
	attrs := make([]slog.Attr, 0, v.NumField())
	for i := range v.NumField() {
		attrs = append(attrs, slog.String(v.Type().Field(i).Name, fmt.Sprintf("%T", v.Field(i).Interface())))
	}
	return slog.GroupValue(attrs...)
}

func ApplyFRRTemplateQuirks(q SonicQuirks, cfg types.Conf) *types.Conf {
	cfg.KeepUnprovisionedInterfaces = q.FrrKeepInterfaces.Enabled()
	cfg.DisableNextHopGroups = q.FrrDisableNextHopGroups.Enabled()
	cfg.SetFPMAddress = q.FrrUseDplaneFpmSonic.Enabled()
	return &cfg
}
