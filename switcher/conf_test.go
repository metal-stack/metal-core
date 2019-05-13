package switcher

import (
	"testing"

	"git.f-i-ts.de/cloud-native/metallib/vlan"
	"github.com/stretchr/testify/require"
)

func TestFillVLANIDs(t *testing.T) {
	m := vlan.Mapping{1001: 101001, 1003: 101003}
	tenants := map[string]*Tenant{
		"101001": &Tenant{VNI: 101001},
		"101002": &Tenant{VNI: 101002},
		"101003": &Tenant{VNI: 101003}}
	c := Conf{Tenants: tenants}
	c.FillVLANIDs(m)
	require.Equal(t, uint16(1001), c.Tenants["101001"].VLANID)
	require.Equal(t, uint16(1002), c.Tenants["101002"].VLANID)
	require.Equal(t, uint16(1003), c.Tenants["101003"].VLANID)
}
