package types

import (
	"net/netip"
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

func TestDeriveBootPorts(t *testing.T) {
	tests := []struct {
		name     string
		block    string
		ordinals map[string]int
		ports    []string
		want     map[string]BootPort
		wantErr  bool
	}{
		{
			name:     "no ports",
			block:    "fd00:20:0:100::/56",
			ordinals: map[string]int{},
			ports:    nil,
			want:     map[string]BootPort{},
		},
		{
			name:     "ports get the n-th /64 of the block",
			block:    "fd00:20:0:100::/56",
			ordinals: map[string]int{"Ethernet0": 0, "Ethernet4": 1, "Ethernet8": 2, "Ethernet124": 255},
			ports:    []string{"Ethernet0", "Ethernet8", "Ethernet124"},
			want: map[string]BootPort{
				"Ethernet0":   {Prefix: "fd00:20:0:100::/64", Address: "fd00:20:0:100::1/64"},
				"Ethernet8":   {Prefix: "fd00:20:0:102::/64", Address: "fd00:20:0:102::1/64"},
				"Ethernet124": {Prefix: "fd00:20:0:1ff::/64", Address: "fd00:20:0:1ff::1/64"},
			},
		},
		{
			name:     "unmasked block is masked",
			block:    "fd00:20:0:1ff::/56",
			ordinals: map[string]int{"swp1": 1},
			ports:    []string{"swp1"},
			want: map[string]BootPort{
				"swp1": {Prefix: "fd00:20:0:101::/64", Address: "fd00:20:0:101::1/64"},
			},
		},
		{
			name:     "a /64 block only allows ordinal 0",
			block:    "fd00:20::/64",
			ordinals: map[string]int{"swp1": 0, "swp2": 1},
			ports:    []string{"swp1", "swp2"},
			wantErr:  true,
		},
		{
			name:     "ordinal exceeds block",
			block:    "fd00:20:0:100::/56",
			ordinals: map[string]int{"swp1": 256},
			ports:    []string{"swp1"},
			wantErr:  true,
		},
		{
			name:     "unknown port",
			block:    "fd00:20:0:100::/56",
			ordinals: map[string]int{},
			ports:    []string{"swp1"},
			wantErr:  true,
		},
		{
			name:     "ipv4 block is rejected",
			block:    "10.0.0.0/8",
			ordinals: map[string]int{"swp1": 0},
			ports:    []string{"swp1"},
			wantErr:  true,
		},
		{
			name:     "/0 block is rejected",
			block:    "::/0",
			ordinals: map[string]int{"swp1": 0},
			ports:    []string{"swp1"},
			wantErr:  true,
		},
		{
			name:     "block longer than /64 is rejected",
			block:    "fd00:20::/72",
			ordinals: map[string]int{"swp1": 0},
			ports:    []string{"swp1"},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveBootPorts(netip.MustParsePrefix(tt.block), tt.ordinals, tt.ports)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DeriveBootPorts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("DeriveBootPorts() diff = %s", diff)
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
