package observe_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/sonictest"
)

// leaf02 with the l3 vni of Vrf20 and the machine port Ethernet3 in it
var leaf02 = sonictest.Data{
	"CONFIG_DB": {
		"VRF": test.StringMap{
			"Vrf20": test.StringMap{"fallback": "false", "vni": "20"},
		},
		"INTERFACE": test.StringMap{
			"Ethernet3": test.StringMap{"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf20"},
		},
		"PORT": test.StringMap{
			"Ethernet3": test.StringMap{"admin_status": "up", "mtu": "9000"},
		},
	},
	"STATE_DB": {
		"VRF_TABLE": test.StringMap{
			"Vrf20": test.StringMap{"state": "ok"},
		},
		"INTERFACE_TABLE": test.StringMap{
			"Ethernet3": test.StringMap{"vrf": "Vrf20"},
		},
	},
	"APPL_DB": {
		"ROUTE_TABLE": test.StringMap{
			"Vrf20": test.StringMap{
				"10.128.0.1": test.StringMap{"ifname": "Ethernet3", "nexthop": "fe80::1"},
			},
			"10.0.0.0/24": test.StringMap{"ifname": "Ethernet120", "nexthop": "fe80::3"},
		},
	},
	"COUNTERS_DB": {
		"COUNTERS_PORT_NAME_MAP": test.StringMap{"Ethernet3": "oid:0x1000000000023"},
		"COUNTERS_RIF_NAME_MAP":  test.StringMap{"Ethernet3": "oid:0x6000000000dab"},
	},
	"ASIC_DB": {
		"ASIC_STATE": test.StringMap{
			"SAI_OBJECT_TYPE_ROUTER_INTERFACE": test.StringMap{
				"oid": test.StringMap{
					"0x6000000000dab": test.StringMap{"SAI_ROUTER_INTERFACE_ATTR_PORT_ID": "oid:0x1000000000023"},
				},
			},
		},
	},
}

var tables = observe.Tables{
	Config: []db.Key{{"VRF"}, {"INTERFACE"}},
	State:  []db.Key{{"VRF_TABLE"}, {"INTERFACE_TABLE"}},
	Asic:   []db.Key{{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE"}},
}

func TestReader_Read(t *testing.T) {
	databases := sonictest.NewDatabases(t, leaf02)
	r := observe.NewReader(databases.DB(t), sonictest.Kernel{"Ethernet3": true})

	s, err := r.Read(t.Context(), tables)
	require.NoError(t, err)

	require.True(t, s.Config.Has(db.Key{"VRF", "Vrf20"}))
	require.True(t, s.Config.Has(db.Key{"INTERFACE", "Ethernet3"}))
	require.False(t, s.Config.Has(db.Key{"PORT", "Ethernet3"}), "only the given tables are read")
	require.True(t, s.State.Has(db.Key{"INTERFACE_TABLE", "Ethernet3"}))
	require.True(t, s.Asic.Has(db.Key{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE", "oid:0x6000000000dab"}))
	require.Equal(t, db.OID("oid:0x6000000000dab"), s.RifOIDs["Ethernet3"])
	require.Equal(t, db.OID("oid:0x1000000000023"), s.PortOIDs["Ethernet3"])
	require.Equal(t, map[string]bool{"Ethernet3": true}, s.AdminUp)

	_, read := s.Routes("Vrf20")
	require.False(t, read, "routes are only read on request")
}

func TestReader_ReadRoutes(t *testing.T) {
	databases := sonictest.NewDatabases(t, leaf02)
	r := observe.NewReader(databases.DB(t), sonictest.Kernel{})

	s, err := r.Read(t.Context(), tables)
	require.NoError(t, err)
	require.NoError(t, r.ReadRoutes(t.Context(), s, []string{"Vrf20", ""}))

	routes, read := s.Routes("Vrf20")
	require.True(t, read)
	require.Len(t, routes, 1)
	require.Equal(t, "ROUTE_TABLE:Vrf20:10.128.0.1", routes[0].Key)

	routes, read = s.Routes("")
	require.True(t, read)
	require.Len(t, routes, 1, "the default vrf has no routes of other vrfs")
	require.Equal(t, "ROUTE_TABLE:10.0.0.0/24", routes[0].Key)

	_, read = s.Routes("Vrf30")
	require.False(t, read)
}
