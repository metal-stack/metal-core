package types

import (
	"net/netip"
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

func TestFillVLANIDsWithBoot(t *testing.T) {
	c := &Conf{
		Ports: Ports{
			Vrfs: map[string]*Vrf{
				"Vrf104001": {VNI: 104001},
				BootVrfName: {VNI: 104000},
			},
		},
	}
	m := vlan.Mapping{1001: 104000}
	require.NoError(t, c.FillVLANIDs(m))
	require.Equal(t, uint16(1001), c.Ports.Vrfs[BootVrfName].VLANID, "existing mapping must be reused")
	require.Equal(t, uint16(1002), c.Ports.Vrfs["Vrf104001"].VLANID)
	require.Equal(t, uint32(104001), m[1002])

	c.Ports.Vrfs[BootVrfName].VNI = 104001
	require.NoError(t, c.FillVLANIDs(m))
	require.Equal(t, c.Ports.Vrfs["Vrf104001"].VLANID, c.Ports.Vrfs[BootVrfName].VLANID)
}

func TestNewWithoutDownPortsKeepsBootPortsOfUpPorts(t *testing.T) {
	c := &Conf{
		Ports: Ports{
			Unprovisioned: map[string]*UnprovisionedPort{
				"swp1": {Port: "swp1", BootPrefix: netip.MustParsePrefix("fd00:20:0:100::/64")},
				"swp2": {Port: "swp2", BootPrefix: netip.MustParsePrefix("fd00:20:0:101::/64")},
			},
			AdminStatus: map[string]PortStatus{"swp2": PortStatusDown},
			Vrfs:        map[string]*Vrf{BootVrfName: {VNI: 104000, VLANID: 1001}},
		},
	}
	got := c.NewWithoutDownPorts()
	require.Equal(t, c.Ports.Vrfs[BootVrfName], got.Ports.Vrfs[BootVrfName])
	require.NotSame(t, c.Ports.Vrfs[BootVrfName], got.Ports.Vrfs[BootVrfName])
	require.Equal(t, map[string]*UnprovisionedPort{
		"swp1": {Port: "swp1", BootPrefix: netip.MustParsePrefix("fd00:20:0:100::/64")},
	}, got.Ports.Unprovisioned)
	require.NotSame(t, c.Ports.Unprovisioned["swp1"], got.Ports.Unprovisioned["swp1"])
	require.Len(t, c.Ports.Unprovisioned, 2, "original must be untouched")
}
