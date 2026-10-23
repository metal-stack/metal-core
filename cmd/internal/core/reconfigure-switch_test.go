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
			},
			{
				Name: "Ethernet1",
				Vrf:  new("vrf104001"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{"10.240.0.0/12"},
				},
			},
			{
				Name: "Ethernet2",
				Vrf:  new("default"),
			},
			{
				Name: "Ethernet3",
			},
			{
				Name: "Ethernet4",
				Vrf:  new("vrf200"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{"10.132.0.0/24", "fe80::10e2::/64"},
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
		LogLevel:      "warnings",
		Loopback:      "10.0.0.1",
		MetalCoreCIDR: "10.255.0.1/24",
		ASN:           420000001,
		Name:          s.Id,
		Ports: types.Ports{
			"Ethernet0": {
				AdminStatus: types.PortStatusUp,
				MTU:         9000,
				VLAN:        new(uint16(4000)),
			},
			"Ethernet1": {
				Vrf:         new("Vrf104001"),
				AdminStatus: types.PortStatusUp,
				MTU:         9000,
			},
			"Ethernet2": {
				Vrf:         new("default"),
				AdminStatus: types.PortStatusUp,
				MTU:         9216,
			},
			"Ethernet3": {
				AdminStatus: types.PortStatusUp,
				MTU:         9000,
				VLAN:        new(uint16(4000)),
			},
			"Ethernet4": {
				Vrf:         new("Vrf200"),
				AdminStatus: types.PortStatusUp,
				MTU:         9000,
			},
			"Ethernet120": {
				AdminStatus: types.PortStatusUp,
				MTU:         9216,
			},
			"Ethernet124": {
				AdminStatus: types.PortStatusUp,
				MTU:         9216,
			},
		},
		Vrfs: types.Vrfs{
			"Vrf104001": {
				Filter: types.Filter{
					IPPrefixLists: []types.IPPrefixList{
						{
							AddressFamily: types.AddressFamilyIPv4,
							Name:          "Vrf104001-in-prefixes",
							Spec:          "permit 10.240.0.0/12 le 32",
						},
					},
					RouteMaps: []types.RouteMap{
						{
							Name:    "Vrf104001-in",
							Entries: []string{"match ip address prefix-list Vrf104001-in-prefixes"},
							Policy:  types.PolicyPermit,
							Order:   10,
						},
					},
				},
				VNI:   104001,
				Cidrs: []string{"10.240.0.0/12"},
				Has4:  true,
			},
			"Vrf200": {
				Filter: types.Filter{
					IPPrefixLists: []types.IPPrefixList{

						{
							AddressFamily: types.AddressFamilyIPv4,
							Name:          "Vrf200-in-prefixes",
							Spec:          "permit 10.132.0.0/24 le 32",
						},
						{
							AddressFamily: types.AddressFamilyIPv6,
							Name:          "Vrf200-in6-prefixes",
							Spec:          "permit fe80::10e2::/64 le 128",
						},
					},
					RouteMaps: []types.RouteMap{
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
				},
				VNI:   200,
				Cidrs: []string{"10.132.0.0/24", "fe80::10e2::/64"},
				Has4:  true,
				Has6:  true,
			},
		},
		AdditionalMgmtRoutes: []string{"100.1.1.0/24"},
		PXEVlanID:            4000,
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
