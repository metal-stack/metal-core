package object

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/sonictest"
)

// read reads the snapshot of the data like the reconciler does, with the routes of the vrfs.
func read(t *testing.T, data sonictest.Data, routeVrfs ...string) *observe.Snapshot {
	t.Helper()

	databases := sonictest.NewDatabases(t, data)
	r := observe.NewReader(databases.DB(t), sonictest.Kernel{})
	s, err := r.Read(t.Context(), Tables)
	require.NoError(t, err)
	require.NoError(t, r.ReadRoutes(t.Context(), s, routeVrfs))
	return s
}

// frr uses the vrfs and the ports, keyed by vrf and port like "Vrf46|Ethernet1".
type frr struct {
	vrfs  []string
	ports []string
}

func (f frr) UsesVrf(vrf string) bool {
	return slices.Contains(f.vrfs, vrf)
}

func (f frr) UsesPort(vrf, port string) bool {
	return slices.Contains(f.ports, vrf+"|"+port)
}

func requireDone(t *testing.T, s Status) {
	t.Helper()
	require.True(t, s.Done(), "pending: %s", s)
}

func requirePending(t *testing.T, s Status, contains string) {
	t.Helper()
	require.False(t, s.Done())
	require.Contains(t, s.String(), contains)
}

// leaf02 (Edgecore SONiC) with the tenant vrf Vrf20, whose l3 vni is mapped to Vlan1002, and the machine port Ethernet3 in it.
func leaf02() sonictest.Data {
	return sonictest.Data{
		"CONFIG_DB": {
			"VXLAN_TUNNEL": test.StringMap{"vtep": test.StringMap{"src_ip": "10.0.0.12"}},
			"VRF":          test.StringMap{"Vrf20": test.StringMap{"fallback": "false", "vni": "20"}},
			"VLAN": test.StringMap{
				"Vlan1002": test.StringMap{"vlanid": "1002"},
				"Vlan4000": test.StringMap{"vlanid": "4000"},
			},
			"SUPPRESS_VLAN_NEIGH": test.StringMap{"Vlan1002": test.StringMap{"suppress": "on"}},
			"VLAN_INTERFACE": test.StringMap{
				"Vlan1002":               test.StringMap{"vrf_name": "Vrf20"},
				"Vlan4000":               test.StringMap{},
				"Vlan4000|10.255.2.1/24": test.StringMap{},
			},
			"VXLAN_TUNNEL_MAP": test.StringMap{
				"vtep": test.StringMap{"map_20_Vlan1002": test.StringMap{"vlan": "Vlan1002", "vni": "20"}},
			},
			"INTERFACE": test.StringMap{
				"Ethernet3":   test.StringMap{"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf20"},
				"Ethernet120": test.StringMap{"ipv6_use_link_local_only": "enable"},
			},
			"VLAN_MEMBER": test.StringMap{
				"Vlan4000": test.StringMap{"Ethernet0": test.StringMap{"tagging_mode": "untagged"}},
			},
		},
		"STATE_DB": {
			"VRF_TABLE":                 test.StringMap{"Vrf20": test.StringMap{"state": "ok"}},
			"VRF_OBJECT_TABLE":          test.StringMap{"Vrf20": test.StringMap{"state": "ok", "oid": "0x3000000000ba9"}},
			"SUPPRESS_VLAN_NEIGH_TABLE": test.StringMap{"Vlan1002": test.StringMap{"netdev": "vtep-1002"}},
			"VLAN_TABLE": test.StringMap{
				"Vlan1002": test.StringMap{"state": "ok"},
				"Vlan4000": test.StringMap{"state": "ok"},
			},
			"INTERFACE_TABLE": test.StringMap{
				"Vlan1002":    test.StringMap{"vrf": "Vrf20"},
				"Vlan4000":    test.StringMap{"vrf": ""},
				"Ethernet3":   test.StringMap{"vrf": "Vrf20"},
				"Ethernet120": test.StringMap{"vrf": ""},
			},
			"VLAN_MEMBER_TABLE": test.StringMap{
				"Vlan4000": test.StringMap{"Ethernet0": test.StringMap{"state": "ok"}},
			},
		},
		"APPL_DB": {
			"VRF_TABLE":              test.StringMap{"Vrf20": test.StringMap{"fallback": "false", "vni": "20"}},
			"VXLAN_VRF_TABLE":        test.StringMap{"vtep": test.StringMap{"evpn_map_20_Vrf20": test.StringMap{"vni": "20", "vrf": "Vrf20"}}},
			"VXLAN_TUNNEL_MAP_TABLE": test.StringMap{"vtep": test.StringMap{"map_20_Vlan1002": test.StringMap{"vlan": "Vlan1002", "vni": "20"}}},
			"INTF_TABLE":             test.StringMap{"Vlan1002": test.StringMap{"vrf_name": "Vrf20"}},
			"ROUTE_TABLE": test.StringMap{
				"Vrf20": test.StringMap{
					"fe80::/64":  test.StringMap{"ifname": "Ethernet3", "nexthop": "::"},
					"10.128.0.1": test.StringMap{"ifname": "Ethernet3", "nexthop": "fe80::527c:6fff:fe3e:8d90"},
				},
			},
		},
		"COUNTERS_DB": {
			"COUNTERS_PORT_NAME_MAP": test.StringMap{
				"Ethernet0": "oid:0x1000000000020",
				"Ethernet3": "oid:0x1000000000023",
			},
			"COUNTERS_RIF_NAME_MAP": test.StringMap{"Ethernet3": "oid:0x6000000000dab"},
		},
		"ASIC_DB": {
			"ASIC_STATE": test.StringMap{
				"SAI_OBJECT_TYPE_ROUTER_INTERFACE": test.StringMap{
					"oid": test.StringMap{"0x6000000000dab": test.StringMap{"SAI_ROUTER_INTERFACE_ATTR_PORT_ID": "oid:0x1000000000023"}},
				},
				"SAI_OBJECT_TYPE_BRIDGE_PORT": test.StringMap{
					"oid": test.StringMap{"0x3a000000000daa": test.StringMap{"SAI_BRIDGE_PORT_ATTR_PORT_ID": "oid:0x1000000000020"}},
				},
			},
		},
	}
}

// without removes the entries from the database, by their key in the database.
func without(data sonictest.Data, database string, keys ...string) sonictest.Data {
	for _, key := range keys {
		deleteKey(data[database], key)
	}
	return data
}

func deleteKey(m test.StringMap, key string) {
	for k, v := range m {
		if k == key {
			delete(m, k)
			return
		}
		if rest, ok := strings.CutPrefix(key, k); ok && len(rest) > 1 && strings.ContainsAny(rest[:1], ":|") {
			if sub, ok := v.(test.StringMap); ok {
				deleteKey(sub, rest[1:])
			}
		}
	}
}
