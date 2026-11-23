package desired

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/vrfentries"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// cfg assigns the ports the roles of the leaf01 test configuration of the core package.
var cfg = &types.Conf{
	PXEVlanID: 4000,
	Ports: types.Ports{
		Underlay:      []string{"Ethernet120", "Ethernet124"},
		Unprovisioned: []string{"Ethernet0", "Ethernet1", "Ethernet2", "Ethernet3"},
		Firewalls:     map[string]*types.Firewall{"Ethernet4": {Port: "Ethernet4"}},
		Vrfs: types.Vrfs{
			"Vrf100": {VNI: 100, VLANID: 1001, Neighbors: []string{"Ethernet6"}},
			"Vrf200": {VNI: 200, VLANID: 1002, Neighbors: []string{"Ethernet5"}},
		},
		AdminStatus: map[string]types.PortStatus{
			"Ethernet0":   types.PortStatusUp,
			"Ethernet1":   types.PortStatusDown,
			"Ethernet2":   types.PortStatusUp,
			"Ethernet3":   types.PortStatusUp,
			"Ethernet4":   types.PortStatusUp,
			"Ethernet5":   types.PortStatusUp,
			"Ethernet6":   types.PortStatusDown,
			"Ethernet120": types.PortStatusUp,
			"Ethernet124": types.PortStatusUp,
		},
	},
}

func entries(sw *Switch) map[string]db.Val {
	m := map[string]db.Val{}
	for _, o := range sw.Objects {
		m[strings.Join(o.Key(), "|")] = o.Fields()
	}
	return m
}

// TestBuild checks the entries against the CONFIG_DB the imperative applier left for the configuration,
// see leaf01WantConfigDB in the core package.
func TestBuild(t *testing.T) {
	sw, err := Build(cfg, vrfentries.Community{}, "vtep")
	require.NoError(t, err)

	want := map[string]db.Val{
		"INTERFACE|Ethernet120":                  {"ipv6_use_link_local_only": "enable"},
		"INTERFACE|Ethernet124":                  {"ipv6_use_link_local_only": "enable"},
		"INTERFACE|Ethernet4":                    {"ipv6_use_link_local_only": "enable"},
		"INTERFACE|Ethernet5":                    {"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf200"},
		"INTERFACE|Ethernet6":                    {"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf100"},
		"PORT|Ethernet0":                         {"admin_status": "up", "mtu": "9000"},
		"PORT|Ethernet1":                         {"admin_status": "down", "mtu": "9000"},
		"PORT|Ethernet2":                         {"admin_status": "up", "mtu": "9000"},
		"PORT|Ethernet3":                         {"admin_status": "up", "mtu": "9000"},
		"PORT|Ethernet4":                         {"admin_status": "up", "mtu": "9216"},
		"PORT|Ethernet5":                         {"admin_status": "up", "mtu": "9000"},
		"PORT|Ethernet6":                         {"admin_status": "down", "mtu": "9000"},
		"PORT|Ethernet120":                       {"admin_status": "up", "mtu": "9216"},
		"PORT|Ethernet124":                       {"admin_status": "up", "mtu": "9216"},
		"SUPPRESS_VLAN_NEIGH|Vlan1001":           {"suppress": "on"},
		"SUPPRESS_VLAN_NEIGH|Vlan1002":           {"suppress": "on"},
		"VLAN_INTERFACE|Vlan1001":                {"vrf_name": "Vrf100"},
		"VLAN_INTERFACE|Vlan1002":                {"vrf_name": "Vrf200"},
		"VLAN_MEMBER|Vlan4000|Ethernet0":         {"tagging_mode": "untagged"},
		"VLAN_MEMBER|Vlan4000|Ethernet1":         {"tagging_mode": "untagged"},
		"VLAN_MEMBER|Vlan4000|Ethernet2":         {"tagging_mode": "untagged"},
		"VLAN_MEMBER|Vlan4000|Ethernet3":         {"tagging_mode": "untagged"},
		"VLAN|Vlan1001":                          {"vlanid": "1001"},
		"VLAN|Vlan1002":                          {"vlanid": "1002"},
		"VRF|Vrf100":                             {"fallback": "false", "vni": "100"},
		"VRF|Vrf200":                             {"fallback": "false", "vni": "200"},
		"VXLAN_TUNNEL_MAP|vtep|map_100_Vlan1001": {"vlan": "Vlan1001", "vni": "100"},
		"VXLAN_TUNNEL_MAP|vtep|map_200_Vlan1002": {"vlan": "Vlan1002", "vni": "200"},
	}
	if diff := cmp.Diff(want, entries(sw)); diff != "" {
		t.Errorf("Build() diff = %s", diff)
	}

	require.Equal(t, []string{
		"Ethernet0", "Ethernet1", "Ethernet120", "Ethernet124", "Ethernet2", "Ethernet3", "Ethernet4", "Ethernet5", "Ethernet6",
	}, sw.Ports)
	require.Len(t, sw.Vrfs["Vrf100"], 5)
	require.Equal(t, db.Key{"INTERFACE", "Ethernet6"}, sw.RouterInterfaces["Ethernet6"].Key())
}

func TestBuild_L3Only(t *testing.T) {
	sw, err := Build(cfg, vrfentries.L3Only{}, "vtep")
	require.NoError(t, err)

	got := entries(sw)
	require.Equal(t, db.Val{"vni": "100", "source_vtep": "vtep"}, got["VRF|Vrf100"])
	require.NotContains(t, got, "VLAN|Vlan1001")
	require.Len(t, sw.Vrfs["Vrf100"], 1)
}

func TestBuild_PortWithTwoRoles(t *testing.T) {
	c := *cfg
	c.Ports.Underlay = []string{"Ethernet120", "Ethernet0"}

	_, err := Build(&c, vrfentries.Community{}, "vtep")
	require.ErrorContains(t, err, "PORT|Ethernet0")
}
