package core

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func TestBuildSwitcherConfig(t *testing.T) {
	c := &Core{
		cidr:                 "10.255.255.2/24",
		partitionID:          "fra-equ01",
		rackID:               "rack01",
		roomID:               "room01",
		asn:                  "420000001",
		loopbackIP:           "10.0.0.1",
		spineUplinks:         []string{"swp31", "swp32"},
		additionalBridgeVIDs: []string{"201-256", "301-356"},
		nos:                  &cumulus.Cumulus{},
	}

	n1 := "swp1"
	m1 := "00:00:00:00:00:01"
	swp1 := &apiv2.SwitchNic{
		Name: n1,
		Mac:  &m1,
	}
	n2 := "swp2"
	m2 := "00:00:00:00:00:02"
	swp2 := &apiv2.SwitchNic{
		Name: n2,
		Mac:  &m2,
		Vrf:  new("vrf104001"),
		BgpFilter: &apiv2.BGPFilter{
			Cidrs: []string{
				"10.240.0.0/12", // pod ipv4 cidrs
			},
		},
	}
	n3 := "swp3"
	m3 := "00:00:00:00:00:03"
	swp3 := &apiv2.SwitchNic{
		Name: n3,
		Mac:  &m3,
		Vrf:  new("default"),
	}
	s := &apiv2.Switch{
		Nics: []*apiv2.SwitchNic{
			swp1,
			swp2,
			swp3,
		},
	}
	actual, err := c.buildSwitcherConfig(t.Context(), s)
	require.NoError(t, err)
	require.NotNil(t, actual)
	expected := &types.Conf{
		LogLevel:      "warnings",
		Loopback:      "10.0.0.1",
		MetalCoreCIDR: "10.255.255.2/24",
		ASN:           420000001,
		Ports: types.Ports{
			AdminStatus:   map[string]types.PortStatus{},
			Underlay:      []string{"swp31", "swp32"},
			Unprovisioned: []string{"swp1"},
			Firewalls: map[string]*types.Firewall{
				"swp3": {
					Port: "swp3",
				},
			},
			Vrfs: map[string]*types.Vrf{"vrf104001": {
				VNI:       104001,
				VLANID:    1001,
				Neighbors: []string{"swp2"},
				Filter: types.Filter{
					IPPrefixLists: []types.IPPrefixList{
						{
							AddressFamily: "ip",
							Name:          "vrf104001-in-prefixes",
							Spec:          "permit 10.240.0.0/12 le 32",
						},
					},
					RouteMaps: []types.RouteMap{
						{
							Name:    "vrf104001-in",
							Entries: []string{"match ip address prefix-list vrf104001-in-prefixes"},
							Policy:  "permit",
							Order:   10,
						},
					},
				},
				Cidrs: []string{"10.240.0.0/12"},
				Has4:  true,
			}},
		},
		AdditionalBridgeVIDs: []string{"201-256", "301-356"},
	}
	require.EqualValues(t, expected, actual)
}

type fakeNOS struct {
	switcher.NOS
	ordinals map[string]int
	err      error
}

func (f *fakeNOS) SanitizeConfig(cfg *types.Conf) {
	cfg.CapitalizeVrfName()
}

func (f *fakeNOS) GetPortOrdinals(context.Context) (map[string]int, error) {
	return f.ordinals, f.err
}

func TestBuildSwitcherConfigL3(t *testing.T) {
	boot, err := NewBootConfig("l3", 104000, "fd00:20:0:100::/56", []string{"fd00:20:ffff::53"})
	require.NoError(t, err)

	c := &Core{
		log:          slog.Default(),
		asn:          "420000001",
		loopbackIP:   "10.0.0.1",
		spineUplinks: []string{"Ethernet120"},
		boot:         boot,
		nos: &fakeNOS{
			ordinals: map[string]int{"Ethernet0": 0, "Ethernet4": 1, "Ethernet8": 2, "Ethernet120": 3},
		},
	}

	s := &apiv2.Switch{
		Nics: []*apiv2.SwitchNic{
			{Name: "Ethernet0"},
			{Name: "Ethernet4", Vrf: new("vrf104001")},
			{Name: "Ethernet8"},
			{Name: "Ethernet120"},
		},
	}

	actual, err := c.buildSwitcherConfig(t.Context(), s)
	require.NoError(t, err)
	require.Equal(t, []string{"Ethernet0", "Ethernet8"}, actual.Ports.Unprovisioned)
	require.NotNil(t, actual.Boot)
	require.Equal(t, "VrfBoot", actual.Boot.Vrf)
	require.Equal(t, uint32(104000), actual.Boot.VNI)
	require.Equal(t, []string{"fd00:20:ffff::53"}, actual.Boot.RDNSS)
	require.NotZero(t, actual.Boot.VLANID, "boot vrf must get a switch-local vlan")
	require.NotEqual(t, actual.Boot.VLANID, actual.Ports.Vrfs["Vrf104001"].VLANID)
	require.Equal(t, map[string]types.BootPort{
		"Ethernet0": {Prefix: "fd00:20:0:100::/64", Address: "fd00:20:0:100::1/64"},
		"Ethernet8": {Prefix: "fd00:20:0:102::/64", Address: "fd00:20:0:102::1/64"},
	}, actual.Boot.Ports)

	// a stale nic unknown to the switch only loses its boot prefix, the switch is still configured
	s.Nics = append(s.Nics, &apiv2.SwitchNic{Name: "Ethernet999"})
	actual, err = c.buildSwitcherConfig(t.Context(), s)
	require.NoError(t, err)
	require.Contains(t, actual.Ports.Unprovisioned, "Ethernet999")
	require.NotContains(t, actual.Boot.Ports, "Ethernet999")
	s.Nics = s.Nics[:len(s.Nics)-1]

	// a boot vni that collides with a tenant vni is rejected
	c.boot.VNI = 104001
	_, err = c.buildSwitcherConfig(t.Context(), s)
	require.ErrorContains(t, err, "boot vni 104001 collides with the vni of vrf Vrf104001")
	c.boot = boot

	// pxe mode does not build a boot config
	c.boot = BootConfig{Mode: BootModePXE}
	actual, err = c.buildSwitcherConfig(t.Context(), s)
	require.NoError(t, err)
	require.Nil(t, actual.Boot)

	// errors of the nos are propagated
	c.boot = boot
	c.nos = &fakeNOS{err: fmt.Errorf("not supported")}
	_, err = c.buildSwitcherConfig(t.Context(), s)
	require.ErrorContains(t, err, "not supported")
}

func TestNewBootConfig(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		vni     uint32
		prefix  string
		rdnss   []string
		wantErr string
	}{
		{name: "pxe needs nothing", mode: "pxe"},
		{name: "l3 valid", mode: "l3", vni: 104000, prefix: "fd00:20:0:1ff::/56", rdnss: []string{"fd00:20:ffff::53"}},
		{name: "unknown mode", mode: "dhcp", wantErr: "unknown boot mode"},
		{name: "l3 without vni", mode: "l3", prefix: "fd00:20::/56", wantErr: "requires a boot vni"},
		{name: "l3 vni out of range", mode: "l3", vni: 1 << 24, prefix: "fd00:20::/56", wantErr: "requires a boot vni"},
		{name: "l3 without prefix", mode: "l3", vni: 1, wantErr: "requires a valid ipv6 boot prefix"},
		{name: "l3 ipv4 prefix", mode: "l3", vni: 1, prefix: "10.0.0.0/8", wantErr: "must be an ipv6 prefix"},
		{name: "l3 prefix too small", mode: "l3", vni: 1, prefix: "fd00:20::/72", wantErr: "must be between /1 and /64"},
		{name: "l3 default route as prefix", mode: "l3", vni: 1, prefix: "::/0", wantErr: "must be between /1 and /64"},
		{name: "l3 invalid rdnss", mode: "l3", vni: 1, prefix: "fd00:20::/56", rdnss: []string{"10.0.0.53"}, wantErr: "must be an ipv6 address"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBootConfig(tt.mode, tt.vni, tt.prefix, tt.rdnss)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, BootMode(tt.mode), got.Mode)
			if tt.mode == "l3" {
				require.Equal(t, "fd00:20:0:100::/56", got.Prefix.String(), "prefix must be masked")
			}
		})
	}
}
