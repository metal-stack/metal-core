package core

import (
	"bytes"
	_ "embed"
	"fmt"
	"testing"
	"text/template"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/go-cmp/cmp"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/templates"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
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
		staticFRRConfig: &types.FRRConfig{
			VRFs: []string{"Vrf300"},
			UnnumberedInterfaces: []*types.Interface{
				{
					Name: "Ethernet6",
					Vrf:  new("Vrf300"),
				},
			},
			BGPs: []*types.BGP{
				{
					PeerGroups: []types.PeerGroup{
						{
							Name: "K3S",
							Commands: []string{
								"remote-as external",
								"timers 2 8",
								"ebgp-multihop 2",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "10.1.1.10",
							PeerGroup: new("K3S"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute connected"},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalCommands: []string{"advertise-all-vni"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "K3S",
									Commands: []string{"activate"},
								},
							},
						},
					},
				},
				{
					Vrf: new("Vrf300"),
					PeerGroups: []types.PeerGroup{
						{
							Name: "K3S",
							Commands: []string{
								"remote-as external",
								"timers 3 9",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet6 interface",
							PeerGroup: new("K3S"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute static"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "K3S",
									Commands: []string{"route-map K3S-in in"},
								},
							},
						},
					},
				},
			},
			RouteMaps: []*types.RouteMap{
				{
					Name:    "K3S-in",
					Entries: []string{"match ip address prefix-list K3S-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
			},
			PrefixLists: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "K3S-in-prefixes",
					Spec:          "permit 10.245.0.0/14 le 32",
				},
			},
		},
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
				Name: "Ethernet5",
				Vrf:  new("vrf200"),
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
		Ports: []types.Port{
			{
				Name:        "Ethernet0",
				AdminStatus: types.PortStatusDown,
				VLAN:        new("Vlan4000"),
				MTU:         9000,
			},
			{
				Name:        "Ethernet1",
				AdminStatus: types.PortStatusUp,
				VRF:         new("Vrf104001"),
				MTU:         9000,
			},
			{
				Name: "Ethernet2",
				MTU:  9216,
			},
			{
				Name: "Ethernet3",
				MTU:  9000,
				VLAN: new("Vlan4000"),
			},
			{
				Name: "Ethernet4",
				VRF:  new("Vrf200"),
				MTU:  9000,
			},
			{
				Name: "Ethernet5",
				VRF:  new("Vrf200"),
				MTU:  9000,
			},
			{
				Name: "Ethernet120",
				MTU:  9216,
			},
			{
				Name: "Ethernet124",
				MTU:  9216,
			},
		},
		Vrfs: []*types.Vrf{
			{
				Name:   "Vrf104001",
				VNI:    104001,
				VLANID: 1001,
			},
			{
				Name:   "Vrf200",
				VNI:    200,
				VLANID: 1002,
			},
			{
				Name:   "Vrf300",
				VNI:    300,
				VLANID: 1003,
			},
		},
		FRRConfig: &types.FRRConfig{
			VRFs: []string{"Vrf104001", "Vrf200", "Vrf300"},
			UnnumberedInterfaces: []*types.Interface{
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
					Name: "Ethernet5",
					Vrf:  new("Vrf200"),
				},
				{
					Name: "Ethernet6",
					Vrf:  new("Vrf300"),
				},
				{
					Name: "Ethernet120",
				},
				{
					Name: "Ethernet124",
				},
			},
			BGPs: []*types.BGP{
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
						{
							Name: "K3S",
							Commands: []string{
								"remote-as external",
								"timers 2 8",
								"ebgp-multihop 2",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet120 interface",
							PeerGroup: new("FABRIC"),
						},
						{
							Name:      "Ethernet124 interface",
							PeerGroup: new("FABRIC"),
						},
						{
							Name:      "Ethernet2 interface",
							PeerGroup: new("FIREWALL"),
						},
						{
							Name:      "10.1.1.10",
							PeerGroup: new("K3S"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{
								"redistribute connected",
								"redistribute connected route-map DENY_MGMT",
								"redistribute kernel route-map DENY_MGMT",
							},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "Ethernet2",
									Commands: []string{"route-map fw-Ethernet2-in in"},
								},
								{
									Name:     "FIREWALL",
									Commands: []string{"allowas-in 2"},
								},
							},
						},
						IPv6Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{
								"redistribute connected route-map DENY_MGMT",
								"redistribute kernel route-map DENY_MGMT",
							},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "FIREWALL",
									Commands: []string{"activate", "allowas-in 2"},
								},
								{
									Name:     "Ethernet2",
									Commands: []string{"route-map fw-Ethernet2-in in"},
								},
							},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalCommands: []string{
								"advertise-all-vni",
							},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "Ethernet2",
									Commands: []string{"route-map fw-Ethernet2-vni out"},
								},
								{
									Name:     "FABRIC",
									Commands: []string{"activate", "allowas-in 2"},
								},
								{
									Name:     "FIREWALL",
									Commands: []string{"activate", "allowas-in 2"},
								},
								{
									Name:     "K3S",
									Commands: []string{"activate"},
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
							GlobalCommands: []string{
								"redistribute connected",
							},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name: "MACHINE",
									Commands: []string{
										"maximum-prefix 24000",
										"route-map Vrf104001-in in",
									},
								},
							},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalCommands: []string{
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
						{
							Name:      "Ethernet5 interface",
							PeerGroup: new("MACHINE"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{
								"redistribute connected",
							},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name: "MACHINE",
									Commands: []string{
										"maximum-prefix 24000",
										"route-map Vrf200-in in",
									},
								},
							},
						},
						IPv6Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{
								"redistribute connected",
							},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name: "MACHINE",
									Commands: []string{
										"activate",
										"maximum-prefix 24000",
										"route-map Vrf200-in6 in",
									},
								},
							},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalCommands: []string{
								"advertise ipv4 unicast",
								"advertise ipv6 unicast",
							},
						},
					},
				},
				{
					Vrf: new("Vrf300"),
					PeerGroups: []types.PeerGroup{
						{
							Name: "K3S",
							Commands: []string{
								"remote-as external",
								"timers 3 9",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet6 interface",
							PeerGroup: new("K3S"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute static"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "K3S",
									Commands: []string{"route-map K3S-in in"},
								},
							},
						},
					},
				},
			},
			RouteMaps: []*types.RouteMap{
				{
					Name:    "DENY_MGMT",
					Entries: []string{"match interface eth0"},
					Policy:  types.PolicyDeny,
					Order:   10,
				},
				{
					Name:   "DENY_MGMT",
					Policy: types.PolicyPermit,
					Order:  20,
				},
				{
					Name:    "K3S-in",
					Entries: []string{"match ip address prefix-list K3S-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
				{
					Name:    "Vrf104001-in",
					Entries: []string{"match ip address prefix-list Vrf104001-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
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
			},
			PrefixLists: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "K3S-in-prefixes",
					Spec:          "permit 10.245.0.0/14 le 32",
				},
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf104001-in-prefixes",
					Spec:          "permit 10.240.0.0/12 le 32",
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
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "fw-Ethernet2-in-prefixes",
					Spec:          "permit 10.240.1.0/24 le 32",
				},
			},
		},
	}

	if diff := cmp.Diff(expected, actual); diff != "" {
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

func Test_mergeVRFs(t *testing.T) {
	tests := []struct {
		name  string
		vrfs1 []string
		vrfs2 []string
		want  []string
	}{
		{
			name:  "merge vrfs",
			vrfs1: []string{"vrf200", "vrf300", "vrf100"},
			vrfs2: []string{"vrf100", "vrf400", "vrf200"},
			want:  []string{"vrf100", "vrf200", "vrf300", "vrf400"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeVRFs(tt.vrfs1, tt.vrfs2)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergeVRFs() diff = %s", diff)
			}
		})
	}
}

func Test_mergeInterfaces(t *testing.T) {
	tests := []struct {
		name    string
		ifs1    []*types.Interface
		ifs2    []*types.Interface
		want    []*types.Interface
		wantErr error
	}{
		{
			name: "merge interfaces",
			ifs1: []*types.Interface{
				{
					Name: "Ethernet1",
					Vrf:  new("vrf100"),
				},
				{
					Name: "Ethernet2",
				},
			},
			ifs2: []*types.Interface{
				{
					Name: "Ethernet0",
					Vrf:  new("vrf100"),
				},
			},
			want: []*types.Interface{
				{
					Name: "Ethernet0",
					Vrf:  new("vrf100"),
				},
				{
					Name: "Ethernet1",
					Vrf:  new("vrf100"),
				},
				{
					Name: "Ethernet2",
				},
			},
			wantErr: nil,
		},
		{
			name: "duplicate interface is not allowed",
			ifs1: []*types.Interface{
				{
					Name: "Ethernet0",
					Vrf:  new("vrf100"),
				},
			},
			ifs2: []*types.Interface{
				{
					Name: "Ethernet0",
					Vrf:  new("vrf200"),
				},
				{
					Name: "Ethernet1",
					Vrf:  new("vrf200"),
				},
			},
			wantErr: fmt.Errorf("found duplicate interface Ethernet0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeInterfaces(tt.ifs1, tt.ifs2)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("mergeInterfaces() errors differ = %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergeInterfaces() diff = %s", diff)
			}
		})
	}
}

func Test_mergeRouteMaps(t *testing.T) {
	tests := []struct {
		name    string
		rms1    []*types.RouteMap
		rms2    []*types.RouteMap
		want    []*types.RouteMap
		wantErr error
	}{
		{
			name: "merge routemaps",
			rms1: []*types.RouteMap{
				{
					Name:    "Vrf100-in",
					Entries: []string{"match ip address prefix-list Vrf100-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   20,
				},
			},
			rms2: []*types.RouteMap{
				{
					Name:    "Vrf200-in",
					Entries: []string{"match ip address prefix-list Vrf200-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
				{
					Name:    "Vrf100-in",
					Entries: []string{"match interface eth0"},
					Policy:  types.PolicyDeny,
					Order:   10,
				},
			},
			want: []*types.RouteMap{
				{
					Name:    "Vrf100-in",
					Entries: []string{"match interface eth0"},
					Policy:  types.PolicyDeny,
					Order:   10,
				},
				{
					Name:    "Vrf100-in",
					Entries: []string{"match ip address prefix-list Vrf100-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   20,
				},
				{
					Name:    "Vrf200-in",
					Entries: []string{"match ip address prefix-list Vrf200-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
			},
		},
		{
			name: "duplicate name and order not allowed",
			rms1: []*types.RouteMap{
				{
					Name:    "Vrf100-in",
					Entries: []string{"match ip address prefix-list Vrf100-in-prefixes"},
					Policy:  types.PolicyPermit,
					Order:   10,
				},
			},
			rms2: []*types.RouteMap{
				{
					Name:    "Vrf100-in",
					Entries: []string{"match interface eth0"},
					Policy:  types.PolicyDeny,
					Order:   10,
				},
			},
			wantErr: fmt.Errorf(`duplicate route-map "Vrf100-in" with equal order 10`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeRouteMaps(tt.rms1, tt.rms2)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("mergeRouteMaps() errors differ = %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergeRouteMaps() diff = %s", diff)
			}
		})
	}
}

func Test_mergePrefixLists(t *testing.T) {
	tests := []struct {
		name string
		pfs1 []*types.IPPrefixList
		pfs2 []*types.IPPrefixList
		want []*types.IPPrefixList
	}{
		{
			name: "merge prefix-lists",
			pfs1: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf200-in-prefixes",
					Spec:          "permit 10.1.2.0/24 le 32",
				},
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.1.1.0/24 le 32",
				},
			},
			pfs2: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.2.0.0/16",
				},
			},
			want: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.1.1.0/24 le 32",
				},
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.2.0.0/16",
				},
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf200-in-prefixes",
					Spec:          "permit 10.1.2.0/24 le 32",
				},
			},
		},
		{
			name: "duplicates get ignored",
			pfs1: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.1.1.0/24 le 32",
				},
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.2.0.0/16",
				},
			},
			pfs2: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.1.1.0/24 le 32",
				},
			},
			want: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.1.1.0/24 le 32",
				},
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.2.0.0/16",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergePrefixLists(tt.pfs1, tt.pfs2)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergePrefixLists() diff = %s", diff)
			}
		})
	}
}

func Test_mergeBGPs(t *testing.T) {
	tests := []struct {
		name    string
		bgps1   []*types.BGP
		bgps2   []*types.BGP
		want    []*types.BGP
		wantErr error
	}{
		{
			name: "merge bgps",
			bgps1: []*types.BGP{
				{
					Vrf: new("Vrf100"),
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
							Name:      "Ethernet0 interface",
							PeerGroup: new("MACHINE"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute connected route-map LOOPBACKS"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name: "MACHINE",
									Commands: []string{
										"maximum-prefix 24000",
									},
								},
							},
						},
					},
				},
				{
					PeerGroups: []types.PeerGroup{
						{
							Name: "FABRIC",
							Commands: []string{
								"remote-as external",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet2 interface",
							PeerGroup: new("FABRIC"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute connected"},
						},
					},
				},
			},
			bgps2: []*types.BGP{
				{
					Vrf: new("Vrf100"),
					PeerGroups: []types.PeerGroup{
						{
							Name: "EXTERNAL",
							Commands: []string{
								"remote-as external",
								"timers 2 8",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet1 interface",
							PeerGroup: new("EXTERNAL"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute kernel route-map LOOPBACKS"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name: "EXTERNAL",
									Commands: []string{
										"allowas-in 2",
									},
								},
							},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalCommands: []string{"advertise-all-vni"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "MACHINE",
									Commands: []string{"activate"},
								},
							},
						},
					},
				},
				{
					Vrf: new("Vrf300"),
					PeerGroups: []types.PeerGroup{
						{
							Name: "MACHINE",
							Commands: []string{
								"remote-as external",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet3 interface",
							PeerGroup: new("MACHINE"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv6Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute connected"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "MACHINE",
									Commands: []string{"activate"},
								},
							},
						},
					},
				},
			},
			want: []*types.BGP{
				{
					PeerGroups: []types.PeerGroup{
						{
							Name: "FABRIC",
							Commands: []string{
								"remote-as external",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet2 interface",
							PeerGroup: new("FABRIC"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute connected"},
						},
					},
				},
				{
					Vrf: new("Vrf100"),
					PeerGroups: []types.PeerGroup{
						{
							Name: "EXTERNAL",
							Commands: []string{
								"remote-as external",
								"timers 2 8",
							},
						},
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
							PeerGroup: new("EXTERNAL"),
						},
						{
							Name:      "Ethernet0 interface",
							PeerGroup: new("MACHINE"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv4Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{
								"redistribute connected route-map LOOPBACKS",
								"redistribute kernel route-map LOOPBACKS",
							},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name: "EXTERNAL",
									Commands: []string{
										"allowas-in 2",
									},
								},
								{
									Name: "MACHINE",
									Commands: []string{
										"maximum-prefix 24000",
									},
								},
							},
						},
						L2VPNEVPN: &types.BGPAddressFamily{
							GlobalCommands: []string{"advertise-all-vni"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "MACHINE",
									Commands: []string{"activate"},
								},
							},
						},
					},
				},
				{
					Vrf: new("Vrf300"),
					PeerGroups: []types.PeerGroup{
						{
							Name: "MACHINE",
							Commands: []string{
								"remote-as external",
							},
						},
					},
					Neighbors: []types.BGPNeighbor{
						{
							Name:      "Ethernet3 interface",
							PeerGroup: new("MACHINE"),
						},
					},
					AddressFamilies: types.BGPAddressFamilies{
						IPv6Unicast: &types.BGPAddressFamily{
							GlobalCommands: []string{"redistribute connected"},
							Neighbors: []*types.AddressFamilyNeighbor{
								{
									Name:     "MACHINE",
									Commands: []string{"activate"},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeBGPs(tt.bgps1, tt.bgps2)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("mergeBGPs() errors differ = %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergeBGPs() diff = %s", diff)
			}
		})
	}
}

func Test_mergePeerGroups(t *testing.T) {
	tests := []struct {
		name    string
		pgs1    []types.PeerGroup
		pgs2    []types.PeerGroup
		want    []types.PeerGroup
		wantErr error
	}{
		{
			name: "cannot add peer-group twice",
			pgs1: []types.PeerGroup{
				{
					Name: "FABRIC",
				},
			},
			pgs2: []types.PeerGroup{
				{
					Name: "FABRIC",
				},
			},
			wantErr: fmt.Errorf(`cannot add peer-group "FABRIC" twice`),
		},
		{
			name: "merge peer-groups",
			pgs1: []types.PeerGroup{
				{
					Name: "FABRIC",
					Commands: []string{
						"remote-as external",
					},
				},
				{
					Name: "MACHINE",
					Commands: []string{
						"timers 2 8",
						"ebgp-multihop 2",
					},
				},
			},
			pgs2: []types.PeerGroup{
				{
					Name: "FIREWALL",
					Commands: []string{
						"remote-as external",
					},
				},
			},
			want: []types.PeerGroup{
				{
					Name: "FABRIC",
					Commands: []string{
						"remote-as external",
					},
				},
				{
					Name: "FIREWALL",
					Commands: []string{
						"remote-as external",
					},
				},
				{
					Name: "MACHINE",
					Commands: []string{
						"timers 2 8",
						"ebgp-multihop 2",
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergePeerGroups(tt.pgs1, tt.pgs2)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("mergePeerGroups() errors differ = %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergePeerGroups() diff = %s", diff)
			}
		})
	}
}

func Test_mergeBGPNeighbors(t *testing.T) {
	tests := []struct {
		name    string
		neighs1 []types.BGPNeighbor
		neighs2 []types.BGPNeighbor
		want    []types.BGPNeighbor
		wantErr error
	}{
		{
			name: "cannot add same neighbor multiple times",
			neighs1: []types.BGPNeighbor{
				{
					Name: "Ethernet0 interface",
				},
			},
			neighs2: []types.BGPNeighbor{
				{
					Name: "Ethernet0 interface",
				},
			},
			wantErr: fmt.Errorf(`cannot add neighbor "Ethernet0 interface" twice`),
		},
		{
			name: "merge neighbors",
			neighs1: []types.BGPNeighbor{
				{
					Name:      "Ethernet0 interface",
					PeerGroup: new("MACHINE"),
				},
				{
					Name:      "Ethernet1",
					PeerGroup: new("FABRIC"),
				},
			},
			neighs2: []types.BGPNeighbor{
				{
					Name:      "Ethernet2",
					PeerGroup: new("MACHINE"),
				},
			},
			want: []types.BGPNeighbor{
				{
					Name:      "Ethernet1",
					PeerGroup: new("FABRIC"),
				},
				{
					Name:      "Ethernet0 interface",
					PeerGroup: new("MACHINE"),
				},
				{
					Name:      "Ethernet2",
					PeerGroup: new("MACHINE"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeBGPNeighbors(tt.neighs1, tt.neighs2)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Fatalf("mergeBGPNeighbors() errors differ = %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergeBGPNeighbors() diff = %s", diff)
			}
		})
	}
}

func Test_mergeBGPAddressFamily(t *testing.T) {
	tests := []struct {
		name string
		af1  *types.BGPAddressFamily
		af2  *types.BGPAddressFamily
		want *types.BGPAddressFamily
	}{
		{
			name: "merge address family",
			af1: &types.BGPAddressFamily{
				GlobalCommands: []string{"redistribute connected"},
				Neighbors: []*types.AddressFamilyNeighbor{
					{
						Name:     "FABRIC",
						Commands: []string{"allowas-in 2"},
					},
					{
						Name:     "FIREWALL",
						Commands: []string{"activate"},
					},
				},
			},
			af2: &types.BGPAddressFamily{
				GlobalCommands: []string{"redistribute static route-map DENY_MGMT"},
				Neighbors: []*types.AddressFamilyNeighbor{
					{
						Name:     "FIREWALL",
						Commands: []string{"allowas-in 2"},
					},
					{
						Name:     "MACHINE",
						Commands: []string{"activate"},
					},
				},
			},
			want: &types.BGPAddressFamily{
				GlobalCommands: []string{
					"redistribute connected",
					"redistribute static route-map DENY_MGMT",
				},
				Neighbors: []*types.AddressFamilyNeighbor{
					{
						Name:     "FABRIC",
						Commands: []string{"allowas-in 2"},
					},
					{
						Name:     "FIREWALL",
						Commands: []string{"activate", "allowas-in 2"},
					},
					{
						Name:     "MACHINE",
						Commands: []string{"activate"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeBGPAddressFamily(tt.af1, tt.af2)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mergeBGPAddressFamily() diff = %s", diff)
			}
		})
	}
}

func Test_interfaceSmallerFunc(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{
			name: "one doesn't contain digits",
			a:    "Ethernet",
			b:    "Ethernet0",
			want: -1,
		},
		{
			name: "standard sonic naming schema",
			a:    "Ethernet120",
			b:    "Ethernet2",
			want: 118,
		},
		{
			name: "alternative sonic naming schema fist number differs",
			a:    "Eth1/3",
			b:    "Eth3/2",
			want: -2,
		},
		{
			name: "alternative sonic naming schema second number differs",
			a:    "Eth1/3",
			b:    "Eth1/2",
			want: 1,
		},
		{
			name: "cumulus naming schema no breakout",
			a:    "swp1",
			b:    "swp4",
			want: -3,
		},
		{
			name: "cumulus naming schema first number differs",
			a:    "swp1s0",
			b:    "swp4",
			want: -3,
		},
		{
			name: "cumulus naming schema second number differs",
			a:    "swp2s0",
			b:    "swp2s2",
			want: -2,
		},
		{
			name: "breakout port after its parent",
			a:    "swp1s0",
			b:    "swp1",
			want: 1,
		},
		{
			name: "only one contains digits",
			a:    "Ethernet1",
			b:    "lo",
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := interfaceSmallerFunc(tt.a, tt.b); got != tt.want {
				t.Errorf("interfaceSmallerFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_compareBGPNeighbors(t *testing.T) {
	neighs := []types.BGPNeighbor{
		{Name: "Ethernet10 interface"},
		{Name: "Ethernet2 interface"},
	}

	got, err := mergeBGPNeighbors(nil, neighs)
	require.NoError(t, err)

	want := []types.BGPNeighbor{
		{Name: "Ethernet2 interface"},
		{Name: "Ethernet10 interface"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mergeBGPNeighbors() diff = %s", diff)
	}
}

func Test_mergeBGPsKeepsInputsUntouched(t *testing.T) {
	staticBGPs := func() []*types.BGP {
		return []*types.BGP{
			{
				PeerGroups: []types.PeerGroup{{Name: "K3S"}},
				Neighbors:  []types.BGPNeighbor{{Name: "10.1.1.10", PeerGroup: new("K3S")}},
				AddressFamilies: types.BGPAddressFamilies{
					IPv4Unicast: &types.BGPAddressFamily{
						Neighbors: []*types.AddressFamilyNeighbor{
							{Name: "K3S", Commands: []string{"route-map K3S-in in"}},
						},
					},
				},
			},
		}
	}
	dynamicBGPs := []*types.BGP{
		{
			PeerGroups: []types.PeerGroup{{Name: "FABRIC"}},
			Neighbors:  []types.BGPNeighbor{{Name: "Ethernet120 interface", PeerGroup: new("FABRIC")}},
			AddressFamilies: types.BGPAddressFamilies{
				IPv4Unicast: &types.BGPAddressFamily{
					Neighbors: []*types.AddressFamilyNeighbor{
						{Name: "K3S", Commands: []string{"allowas-in 2"}},
					},
				},
			},
		},
	}

	static := staticBGPs()
	for run := range 2 {
		_, err := mergeBGPs(static, dynamicBGPs)
		require.NoError(t, err, "run %d", run)
	}

	if diff := cmp.Diff(staticBGPs(), static); diff != "" {
		t.Errorf("mergeBGPs() changed its first argument, diff = %s", diff)
	}
}

func Test_buildSwitcherConfigTwoMachinesInSameVrf(t *testing.T) {
	c := &Core{
		asn:             "420000001",
		nos:             &sonic.Sonic{},
		pxeVlanID:       4000,
		staticFRRConfig: &types.FRRConfig{},
	}
	s := &apiv2.Switch{
		Id: "leaf01",
		Nics: []*apiv2.SwitchNic{
			{
				Name:      "Ethernet1",
				Vrf:       new("vrf104001"),
				BgpFilter: &apiv2.BGPFilter{Cidrs: []string{"10.240.0.0/12", "212.1.1.1/32"}},
			},
			{
				Name:      "Ethernet2",
				Vrf:       new("vrf104001"),
				BgpFilter: &apiv2.BGPFilter{Cidrs: []string{"10.240.0.0/12", "212.1.1.2/32"}},
			},
		},
	}

	got, err := c.buildSwitcherConfig(s)
	require.NoError(t, err)

	var machineRouteMaps []*types.RouteMap
	for _, rm := range got.FRRConfig.RouteMaps {
		if rm.Name == "Vrf104001-in" {
			machineRouteMaps = append(machineRouteMaps, rm)
		}
	}
	require.Len(t, machineRouteMaps, 1)

	var specs []string
	for _, pf := range got.FRRConfig.PrefixLists {
		specs = append(specs, pf.Spec)
	}
	require.ElementsMatch(t, []string{
		"permit 10.240.0.0/12 le 32",
		"permit 212.1.1.1/32 le 32",
		"permit 212.1.1.2/32 le 32",
	}, specs)
}

func Test_addVRFRejectsNameWithoutVNI(t *testing.T) {
	for _, name := range []string{"v1", "Vrf", "VrfACPmgmt"} {
		t.Run(name, func(t *testing.T) {
			_, err := addVRF(new(name), nil)
			require.Error(t, err)
		})
	}

	vrfs, err := addVRF(new("Vrf104001"), nil)
	require.NoError(t, err)
	require.Equal(t, []*types.Vrf{{Name: "Vrf104001", VNI: 104001}}, vrfs)
}
