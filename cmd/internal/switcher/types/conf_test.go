package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/vlan"
)

func TestFillVLANIDs(t *testing.T) {
	m := vlan.Mapping{1001: 101001, 1003: 101003}
	vrfs := map[string]*Vrf{
		"101001": {VNI: 101001},
		"101002": {VNI: 101002},
		"101003": {VNI: 101003}}
	c := Conf{
		Ports: Ports{
			Vrfs: vrfs,
		},
	}
	err := c.FillVLANIDs(m)
	require.NoError(t, err)
	require.Equal(t, uint16(1001), c.Ports.Vrfs["101001"].VLANID)
	require.Equal(t, uint16(1002), c.Ports.Vrfs["101002"].VLANID)
	require.Equal(t, uint16(1003), c.Ports.Vrfs["101003"].VLANID)
}
