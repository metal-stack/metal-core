package quirks

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuirkProfiles(t *testing.T) {
	for name, newQuirks := range QuirkProfiles {
		t.Run(name, func(t *testing.T) {
			v := reflect.ValueOf(newQuirks())
			for i := range v.NumField() {
				require.False(t, v.Field(i).IsNil(), "quirk %s is not set", v.Type().Field(i).Name)
			}
		})
	}
}

func TestSonicQuirks_LogValue(t *testing.T) {
	got := QuirkProfiles["asternos"]().LogValue().String()
	require.Equal(t, "[VrfEntries=vrfentries.L3Only PortState=portstate.StateDb FrrUseDplaneFpmSonic=frrfpm.Post202411 FrrDisableNextHopGroups=frrnexthopgroups.Post202411 FrrKeepInterfaces=frrkeepinterfaces.Modern LLDPConfiguration=lldp.AsterNOS]", got)
}
