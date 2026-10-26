package quirks

import (
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/portstate"
)

// NewTestQuirks returns the community_old profile, but reads the port state from the CONFIG_DB.
// It is meant for tests that only run against a CONFIG_DB, without syncd filling the ASIC_DB.
func NewTestQuirks() SonicQuirks {
	q := QuirkProfiles["community_old"]()
	q.PortState = portstate.ConfigDb{}
	return q
}
