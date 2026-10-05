package cumulus

import (
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/stretchr/testify/require"
)

func TestApplyRejectsBootNetwork(t *testing.T) {
	// No appliers are initialized: rejection must happen before any file is written.
	c := &Cumulus{}
	err := c.Apply(t.Context(), &types.Conf{Ports: types.Ports{
		Vrfs: map[string]*types.Vrf{types.BootVrfName: {VNI: 104000}},
	}})
	require.ErrorContains(t, err, "layer 3 boot network is only supported on SONiC")
}
