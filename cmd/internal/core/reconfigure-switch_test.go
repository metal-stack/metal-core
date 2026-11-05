package core

import (
	"bytes"
	_ "embed"
	"testing"
	"text/template"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func TestBuildSwitcherConfig(t *testing.T) {
	c := &Core{
		cidr:                 "10.255.0.1/24",
		partitionID:          "partition1",
		rackID:               "rack01",
		roomID:               "room01",
		asn:                  "420000001",
		loopbackIP:           "10.0.0.1",
		spineUplinks:         []string{"Ethernet120", "Ethernet124"},
		nos:                  &sonic.Sonic{},
		additionalMgmtRoutes: []string{"100.1.1.0/24"},
		ignorePorts:          []string{"Ethernet24"},
		pxeVlanID:            4000,
	}

	s := &apiv2.Switch{
		Id: "leaf01",
		Nics: []*apiv2.SwitchNic{
			{
				Name: "Ethernet0",
				State: &apiv2.NicState{
					Desired: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN.Enum(),
				},
			},
			{
				Name: "Ethernet1",
				State: &apiv2.NicState{
					Desired: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP.Enum(),
				},
				Vrf: new("vrf104001"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{"10.240.0.0/12"},
				},
			},
			{
				Name: "Ethernet2",
				Vrf:  new("default"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{"10.240.1.0/24"},
					Vnis:  []string{"104001", "200"},
				},
			},
			{
				Name: "Ethernet3",
			},
			{
				Name: "Ethernet4",
				Vrf:  new("vrf200"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{"10.132.0.0/24", "fe80:10e2::/64"},
				},
			},
			{
				Name: "Ethernet24",
			},
			{
				Name: "Ethernet120",
			},
			{
				Name: "Ethernet124",
			},
		},
	}

	actual, err := c.buildSwitcherConfig(s)
	require.NoError(t, err)

	expected := &types.Conf{
		LogLevel:             "warnings",
		Loopback:             "10.0.0.1",
		MetalCoreCIDR:        "10.255.0.1/24",
		ASN:                  420000001,
		Name:                 s.Id,
		AdditionalMgmtRoutes: []string{"100.1.1.0/24"},
		PXEVlanID:            4000,
		// FIXME: vrfs and ports missing
		FRRConfig: &types.FRRConfig{
			VRFs: []string{"Vrf104001", "Vrf200"},
			UnnumberedInterfaces: []*types.Interface{
				// FIXME: add multiple neighbors to each vrf
				{
					Name: "Ethernet1",
					Vrf:  new("Vrf104001"),
				},
				{
					Name: "Ethernet2",
				},
				{
					Name: "Ethernet4",
					Vrf:  new("Vrf200"),
				},
				{
					Name: "Ethernet120",
				},
				{
					Name: "Ethernet124",
				},
			},
			BGP: []*types.BGP{
				{
					PeerGroups: []types.PeerGroup{
						{
							Name: "FABRIC",
							Commands: []string{
								"remote-as external",
								"timers 2 8",
							},
						},
						{
							Name: "FIREWALL",
							Commands: []string{
								"remote-as external",
								"timers 2 8",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet2 interface",
							PeerGroup: new("FIREWALL"),
						},
						{
							Name:      "Ethernet120 interface",
							PeerGroup: new("FABRIC"),
						},
						{
							Name:      "Ethernet124 interface",
							PeerGroup: new("FABRIC"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalStatements: []string{
								"redistribute connected route-map DENY_MGMT",
								"redistribute kernel route-map DENY_MGMT",
							},
							Neighbors: []types.AdressFamilyNeighbor{
								{
									Neighbor: "FIREWALL",
									Commands: []string{"allowas-in 2"},
								},
								{
									Neighbor:   "Ethernet2",
									RouteMapIn: "fw-Ethernet2-in",
								},
							},
						},
						IPv6Unicast: &types.BGPAddressFamily{
							GlobalStatements: []string{
								"redistribute connected route-map DENY_MGMT",
								"redistribute kernel route-map DENY_MGMT",
							},
							Neighbors: []types.AdressFamilyNeighbor{
								{
									Neighbor: "FIREWALL",
									Commands: []string{"activate", "allowas-in 2"},
								},
								{
									Neighbor:   "Ethernet2",
									RouteMapIn: "fw-Ethernet2-in",
								},
							},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalStatements: []string{
								"advertise-all-vni",
							},
							Neighbors: []types.AdressFamilyNeighbor{
								{
									Neighbor: "FABRIC",
									Commands: []string{"activate", "allowas-in 2"},
								},
								{
									Neighbor: "FIREWALL",
									Commands: []string{"activate", "allowas-in 2"},
								},
								{
									Neighbor:    "Ethernet2",
									RouteMapOut: "fw-Ethernet2-vni",
								},
							},
						},
					},
				},
				{
					Vrf: new("Vrf104001"),
					PeerGroups: []types.PeerGroup{
						{
							Name: "MACHINE",
							Commands: []string{
								"remote-as external",
								"timers 2 8",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet1 interface",
							PeerGroup: new("MACHINE"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalStatements: []string{
								"redistribute connected",
							},
							Neighbors: []types.AdressFamilyNeighbor{
								{
									Neighbor:   "MACHINE",
									Commands:   []string{"maximum-prefix 24000"},
									RouteMapIn: "Vrf104001-in",
								},
							},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalStatements: []string{
								"advertise ipv4 unicast",
							},
						},
					},
				},
				{
					Vrf: new("Vrf200"),
					PeerGroups: []types.PeerGroup{
						{
							Name: "MACHINE",
							Commands: []string{
								"remote-as external",
								"timers 2 8",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet4 interface",
							PeerGroup: new("MACHINE"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalStatements: []string{
								"redistribute connected",
							},
							Neighbors: []types.AdressFamilyNeighbor{
								{
									Neighbor:   "MACHINE",
									Commands:   []string{"maximum-prefix 24000"},
									RouteMapIn: "Vrf200-in",
								},
							},
						},
						IPv6Unicast: &types.BGPAddressFamily{
							GlobalStatements: []string{
								"redistribute connected",
							},
							Neighbors: []types.AdressFamilyNeighbor{
								{
									Neighbor:   "MACHINE",
									Commands:   []string{"activate", "maximum-prefix 24000"},
									RouteMapIn: "Vrf200-in6",
								},
							},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalStatements: []string{
								"advertise ipv4 unicast",
								"advertise ipv6 unicast",
							},
						},
					},
				},
			},
			RouteMaps: []*types.RouteMap{
				{
					Name:    "Vrf104001-in",
					Entries: []string{"match ip address prefix-list Vrf104001-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
				{
					Name:    "fw-Ethernet2-in",
					Entries: []string{"match ip address prefix-list fw-Ethernet2-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
				{
					Name:    "fw-Ethernet2-vni",
					Entries: []string{"match evpn vni 104001"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
				{
					Name:    "fw-Ethernet2-vni",
					Entries: []string{"match evpn vni 200"},
					Policy:  types.PolicyPermit,
					Order:   11,
				},
				{
					Name:    "Vrf200-in",
					Entries: []string{"match ip address prefix-list Vrf200-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
				{
					Name:    "Vrf200-in6",
					Entries: []string{"match ip address prefix-list Vrf200-in6-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
			},
			PrefixLists: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf104001-in-prefixes",
					Spec:          "permit 10.240.0.0/12 le 32",
				},
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "fw-Ethernet2-in-prefixes",
					Spec:          "permit 10.240.1.0/24 le 32",
				},
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf200-in-prefixes",
					Spec:          "permit 10.132.0.0/24 le 32",
				},
				{
					AddressFamily: types.AddressFamilyIPv6,
					Name:          "Vrf200-in6-prefixes",
					Spec:          "permit fe80:10e2::/64 le 128",
				},
			},
		},
	}

	// VLANID is ignored because it is reserved dynamically and the order in which the map is iterated matters so the test becomes flaky.
	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreFields(types.Vrf{}, "VLANID")); diff != "" {
		t.Errorf("TestBuildSwitcherConfig() diff = %s", diff)
	}
}

//go:embed testdata/leaf01-frr.conf
var wantSonicFRRConf string

func TestSonicConfig(t *testing.T) {
	var (
		dbcfg = &db.Config{
			Databases: map[string]db.Database{
				"APPL_DB": {
					Id:        0,
					Instance:  "redis",
					Separator: ":",
				},
				"ASIC_DB": {
					Id:        1,
					Instance:  "redis",
					Separator: ":",
				},
				"COUNTERS_DB": {
					Id:        2,
					Instance:  "redis",
					Separator: ":",
				},
				"CONFIG_DB": {
					Id:        4,
					Instance:  "redis",
					Separator: "|",
				},
			},
		}
	)

	core := New(testcoreCfgSonic)

	mr := miniredis.RunT(t)
	dbcfg.Instances = map[string]db.Instance{
		"redis": {
			Hostname: mr.Addr(),
		},
	}

	vc, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
		SelectDB:     4,
	})
	require.NoError(t, err)

	err = test.LoadData(t.Context(), vc, leaf01PreviousConfigDB, "|")
	require.NoError(t, err)

	database, err := db.New(dbcfg, db.SchemeRedis)
	require.NoError(t, err)

	redisApplier := redis.NewTestApplier(testcoreLog, database)
	err = redisApplier.Apply(t.Context(), leaf01PreviousCfg)
	require.NoError(t, err)

	switcherCfg, err := core.buildSwitcherConfig(leaf01)
	require.NoError(t, err)

	injectStaticVrfVlanMapping(switcherCfg, map[string]uint16{
		"Vrf100": 1001,
		"Vrf200": 1002,
	})
	injectMgmtGateway(switcherCfg, core.managementGateway)

	err = redisApplier.Apply(t.Context(), switcherCfg)
	require.NoError(t, err)

	configDB, err := test.GetData(t.Context(), vc, "|")
	require.NoError(t, err)

	if diff := cmp.Diff(leaf01WantConfigDB, configDB); diff != "" {
		t.Errorf("TestSonicConfig configDB diff = %s", diff)
	}

	tpl := templates.SonicFrrTemplate("")
	frrConf := renderTemplate(t, tpl, switcherCfg)

	if diff := cmp.Diff(wantSonicFRRConf, frrConf); diff != "" {
		t.Errorf("TestSonicConfig frr.conf diff = %s", diff)
	}
}

//go:embed testdata/leaf02-frr.conf
var wantCumulusFRRConf string

//go:embed testdata/leaf02-interfaces
var wantCumulusInterfaces string

func TestCumulusConfig(t *testing.T) {
	core := New(testcoreCfgCumulus)

	switcherCfg, err := core.buildSwitcherConfig(leaf02)
	require.NoError(t, err)

	injectStaticVrfVlanMapping(switcherCfg, map[string]uint16{
		"vrf100": 1001,
		"vrf200": 1002,
	})
	injectMgmtGateway(switcherCfg, core.managementGateway)

	tpl := templates.InterfacesTemplate("")
	intfConf := renderTemplate(t, tpl, switcherCfg)
	if diff := cmp.Diff(wantCumulusInterfaces, intfConf); diff != "" {
		t.Errorf("TestCumulusConfig interfaces diff = %s", diff)
	}

	tpl = templates.CumulusFrrTemplate("")
	frrConf := renderTemplate(t, tpl, switcherCfg)
	if diff := cmp.Diff(wantCumulusFRRConf, frrConf); diff != "" {
		t.Errorf("TestCumulusConfig frr.conf diff = %s", diff)
	}
}

func renderTemplate(t *testing.T, tpl *template.Template, c *types.Conf) string {
	var b bytes.Buffer
	err := tpl.Execute(&b, c)
	require.NoError(t, err)
	return b.String()
}
