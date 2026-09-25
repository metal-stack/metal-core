package types

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestBootPortFromPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		want    BootPort
		wantErr bool
	}{
		{name: "valid /64", prefix: "fd00:20:0:100::/64", want: BootPort{Prefix: "fd00:20:0:100::/64", Address: "fd00:20:0:100::1/64"}},
		{name: "unmasked prefix is masked", prefix: "fd00:20:0:100::ff/64", want: BootPort{Prefix: "fd00:20:0:100::/64", Address: "fd00:20:0:100::1/64"}},
		{name: "/56 is rejected", prefix: "fd00:20:0:100::/56", wantErr: true},
		{name: "/72 is rejected", prefix: "fd00:20:0:100::/72", wantErr: true},
		{name: "ipv4 is rejected", prefix: "10.0.0.0/24", wantErr: true},
		{name: "ipv4 in ipv6 is rejected", prefix: "::ffff:10.0.0.0/64", wantErr: true},
		{name: "garbage is rejected", prefix: "not-a-prefix", wantErr: true},
		{name: "empty is rejected", prefix: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BootPortFromPrefix(tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Fatalf("BootPortFromPrefix() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BootPortFromPrefix() diff = %s", diff)
			}
		})
	}
}

func TestFillVLANIDsWithBoot(t *testing.T) {
	c := &Conf{
		Ports: Ports{
			Vrfs: map[string]*Vrf{"Vrf104001": {VNI: 104001}},
		},
		Boot: &BootConf{Vrf: BootVrfName, VNI: 104000},
	}
	m := vlan.Mapping{1001: 104000}
	require.NoError(t, c.FillVLANIDs(m))
	require.Equal(t, uint16(1001), c.Boot.VLANID, "existing mapping must be reused")
	require.Equal(t, uint16(1002), c.Ports.Vrfs["Vrf104001"].VLANID)
	require.Equal(t, uint32(104001), m[1002])

	c.Boot.VNI = 104001
	require.ErrorContains(t, c.FillVLANIDs(vlan.Mapping{}), "collides with the vni of vrf Vrf104001")
}

func TestNewWithoutDownPortsKeepsBootPortsOfUpPorts(t *testing.T) {
	c := &Conf{
		Ports: Ports{
			Unprovisioned: []string{"swp1", "swp2"},
			AdminStatus:   map[string]PortStatus{"swp2": PortStatusDown},
		},
		Boot: &BootConf{
			Vrf: BootVrfName,
			Ports: map[string]BootPort{
				"swp1": {Prefix: "fd00:20:0:100::/64", Address: "fd00:20:0:100::1/64"},
				"swp2": {Prefix: "fd00:20:0:101::/64", Address: "fd00:20:0:101::1/64"},
			},
		},
	}
	got := c.NewWithoutDownPorts()
	require.Equal(t, []string{"swp1"}, got.Ports.Unprovisioned)
	require.Equal(t, map[string]BootPort{"swp1": {Prefix: "fd00:20:0:100::/64", Address: "fd00:20:0:100::1/64"}}, got.Boot.Ports)
	require.Len(t, c.Boot.Ports, 2, "original must be untouched")
}
