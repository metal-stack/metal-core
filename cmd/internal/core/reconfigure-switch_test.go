package core

import (
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
		Mac:  &m1, // nolint:staticcheck
	}
	n2 := "swp2"
	m2 := "00:00:00:00:00:02"
	swp2 := &apiv2.SwitchNic{
		Name: n2,
		Mac:  &m2, // nolint:staticcheck
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
		Mac:  &m3, // nolint:staticcheck
		Vrf:  new("default"),
	}
	s := &apiv2.Switch{
		Nics: []*apiv2.SwitchNic{
			swp1,
			swp2,
			swp3,
		},
	}
	actual, err := c.buildSwitcherConfig(s)
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
}

func (f *fakeNOS) SanitizeConfig(cfg *types.Conf) {
	cfg.CapitalizeVrfName()
}

func TestBuildSwitcherConfigL3(t *testing.T) {
	boot, err := NewBootConfig("l3")
	require.NoError(t, err)

	c := &Core{
		log:          slog.Default(),
		asn:          "420000001",
		loopbackIP:   "10.0.0.1",
		spineUplinks: []string{"Ethernet120"},
		boot:         boot,
		nos:          &fakeNOS{},
	}

	s := &apiv2.Switch{
		Partition: "partition-1",
		BootVni:   new(uint32(104000)),
		BootRdnss: []string{"fd00:20:ffff::53"},
		Nics: []*apiv2.SwitchNic{
			{Name: "Ethernet0", BootPrefix: new("fd00:20::/64")},
			{Name: "Ethernet4", Vrf: new("vrf104001"), BootPrefix: new("fd00:20:0:1::/64")},
			{Name: "Ethernet8", BootPrefix: new("fd00:20:0:2::/64")},
			{Name: "Ethernet12"}, // no prefix assigned yet
			{Name: "Ethernet16", BootPrefix: new("fd00:20::/56")}, // invalid prefix
			{Name: "Ethernet120", BootPrefix: new("fd00:20:0:3::/64")},
		},
	}

	actual, err := c.buildSwitcherConfig(s)
	require.NoError(t, err)
	require.Equal(t, []string{"Ethernet0", "Ethernet8", "Ethernet12", "Ethernet16"}, actual.Ports.Unprovisioned)
	require.NotNil(t, actual.Boot)
	require.Equal(t, "VrfBoot", actual.Boot.Vrf)
	require.Equal(t, uint32(104000), actual.Boot.VNI)
	require.Equal(t, []string{"fd00:20:ffff::53"}, actual.Boot.RDNSS)
	require.NotZero(t, actual.Boot.VLANID, "boot vrf must get a switch-local vlan")
	require.NotEqual(t, actual.Boot.VLANID, actual.Ports.Vrfs["Vrf104001"].VLANID)
	require.Equal(t, map[string]types.BootPort{
		"Ethernet0": {Prefix: "fd00:20::/64", Address: "fd00:20::1/64"},
		"Ethernet8": {Prefix: "fd00:20:0:2::/64", Address: "fd00:20:0:2::1/64"},
	}, actual.Boot.Ports, "ports without or with an invalid boot prefix are left out")

	// a boot VNI that collides with a tenant VNI is rejected
	s.BootVni = new(uint32(104001))
	_, err = c.buildSwitcherConfig(s)
	require.ErrorContains(t, err, "boot vni 104001 collides with the vni of vrf Vrf104001")

	// RDNSS addresses must be IPv6 addresses
	s.BootRdnss = []string{"10.0.0.53"}
	_, err = c.buildSwitcherConfig(s)
	require.ErrorContains(t, err, `boot rdnss address "10.0.0.53" must be an ipv6 address`)
	s.BootRdnss = []string{"dns"}
	_, err = c.buildSwitcherConfig(s)
	require.ErrorContains(t, err, `invalid boot rdnss address "dns"`)
	s.BootRdnss = nil

	// without a boot network in the partition, the switch cannot be configured in L3 mode
	s.BootVni = nil
	_, err = c.buildSwitcherConfig(s)
	require.ErrorContains(t, err, "requires a boot network in partition partition-1")

	// PXE mode does not build a boot config
	c.boot = BootConfig{Mode: BootModePXE}
	actual, err = c.buildSwitcherConfig(s)
	require.NoError(t, err)
	require.Nil(t, actual.Boot)
}
