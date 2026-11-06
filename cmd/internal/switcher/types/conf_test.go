package types

import (
	"testing"

	"github.com/go-jose/go-jose/v4/testutils/require"
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
)

func TestFillVLANIDs(t *testing.T) {
	var (
		m = vlan.Mapping{1001: 101001, 1003: 101003}

		vrfs = []*Vrf{
			{
				Name: "vrf101001",
				VNI:  101001,
			},
			{
				Name: "vrf101002",
				VNI:  101002,
			},
			{
				Name: "vrf101003",
				VNI:  101003,
			},
		}

		c = Conf{
			Vrfs: vrfs,
		}
	)

	err := c.FillVLANIDs(m)
	require.NoError(t, err)

	require.Equal(t, uint16(1001), c.Vrfs[0].VLANID)
	require.Equal(t, uint16(1002), c.Vrfs[1].VLANID)
	require.Equal(t, uint16(1003), c.Vrfs[2].VLANID)
}
