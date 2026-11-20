package portstate

import (
	"log/slog"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// newTestDB loads the data into the databases, keyed by their name.
func newTestDB(t *testing.T, data map[string]test.StringMap) *db.DB {
	t.Helper()

	mr := miniredis.RunT(t)
	cfg := &db.Config{
		Databases: map[string]db.Database{
			"APPL_DB":     {Id: 0, Instance: "redis", Separator: ":"},
			"ASIC_DB":     {Id: 1, Instance: "redis", Separator: ":"},
			"COUNTERS_DB": {Id: 2, Instance: "redis", Separator: ":"},
			"CONFIG_DB":   {Id: 4, Instance: "redis", Separator: "|"},
			"STATE_DB":    {Id: 6, Instance: "redis", Separator: "|"},
		},
		Instances: map[string]db.Instance{
			"redis": {Hostname: mr.Addr()},
		},
	}

	for name, d := range data {
		database := cfg.Databases[name]
		vc, err := valkey.NewClient(valkey.ClientOption{
			InitAddress:  []string{mr.Addr()},
			DisableCache: true,
			SelectDB:     database.Id,
		})
		require.NoError(t, err)
		require.NoError(t, test.LoadData(t.Context(), vc, d, database.Separator))
		vc.Close()
	}

	sonicDb, err := db.New(cfg, slog.Default(), db.SchemeRedis)
	require.NoError(t, err)
	return sonicDb
}

func pending(t *testing.T, conds []db.Condition) []string {
	t.Helper()
	p, err := db.Pending(t.Context(), conds...)
	require.NoError(t, err)
	return p
}

// Ethernet0 is routed, Ethernet1 is bridged in Vlan4000 and Ethernet2 is neither.
var testData = map[string]test.StringMap{
	"COUNTERS_DB": {
		"COUNTERS_PORT_NAME_MAP": test.StringMap{
			"Ethernet0": "oid:0x1000000000020",
			"Ethernet1": "oid:0x1000000000021",
			"Ethernet2": "oid:0x1000000000022",
		},
		"COUNTERS_RIF_NAME_MAP": test.StringMap{
			"Ethernet0": "oid:0x6000000000daa",
		},
	},
	"ASIC_DB": {
		"ASIC_STATE": test.StringMap{
			"SAI_OBJECT_TYPE_ROUTER_INTERFACE": test.StringMap{
				"oid": test.StringMap{
					"0x6000000000daa": test.StringMap{
						"SAI_ROUTER_INTERFACE_ATTR_PORT_ID": "oid:0x1000000000020",
					},
				},
			},
			"SAI_OBJECT_TYPE_BRIDGE_PORT": test.StringMap{
				"oid": test.StringMap{
					"0x3a000000000daa": test.StringMap{
						"SAI_BRIDGE_PORT_ATTR_PORT_ID": "oid:0x1000000000021",
					},
				},
			},
		},
	},
	"STATE_DB": {
		"INTERFACE_TABLE": test.StringMap{
			"Ethernet0": test.StringMap{"vrf": "Vrf46"},
		},
		"VLAN_TABLE": test.StringMap{
			"Vlan4000": test.StringMap{"state": "ok"},
		},
		"VLAN_MEMBER_TABLE": test.StringMap{
			"Vlan4000|Ethernet1": test.StringMap{"state": "ok"},
		},
	},
}

func TestPortStateCheckers(t *testing.T) {
	checker := AsicDb{}.NewChecker(slog.Default(), newTestDB(t, testData))
	require.NoError(t, checker.Refresh(t.Context()))

	require.NotEmpty(t, pending(t, checker.NotRouted("Ethernet0")))
	require.Empty(t, pending(t, checker.RoutedIn("Ethernet0", "Vrf46")))
	require.Empty(t, pending(t, checker.NotRouted("Ethernet2")))
	require.NotEmpty(t, pending(t, checker.RoutedIn("Ethernet2", "Vrf46")))

	require.NotEmpty(t, pending(t, checker.NotBridged("Ethernet1")))
	require.Empty(t, pending(t, checker.BridgedIn("Ethernet1", "Vlan4000")))
	require.Empty(t, pending(t, checker.NotBridged("Ethernet2")))
	require.NotEmpty(t, pending(t, checker.BridgedIn("Ethernet2", "Vlan4000")))
}

func TestKernelAdminStatus(t *testing.T) {
	require.Empty(t, pending(t, []db.Condition{kernelAdminStatus("lo", types.PortStatusUp)}))
	require.Len(t, pending(t, []db.Condition{kernelAdminStatus("lo", types.PortStatusDown)}), 1)
}

// leaf03 after the vrf teardown race: intfmgrd removed the interface, but orchagent still holds its router interface.
func TestAsicDb_RouterInterfaceLeft(t *testing.T) {
	data := map[string]test.StringMap{
		"COUNTERS_DB": testData["COUNTERS_DB"],
		"ASIC_DB":     testData["ASIC_DB"],
	}
	checker := AsicDb{}.NewChecker(slog.Default(), newTestDB(t, data))
	require.NoError(t, checker.Refresh(t.Context()))

	require.Equal(t, []string{
		"ASIC_DB router interface oid:0x6000000000daa of Ethernet0 absent",
	}, pending(t, checker.NotRouted("Ethernet0")))
	require.Equal(t, []string{
		"STATE_DB INTERFACE_TABLE|Ethernet0 in Vrf46 (absent)",
	}, pending(t, checker.RoutedIn("Ethernet0", "Vrf46")))
}

// leaf03 Ethernet1 after it was moved into Vrf46 on top of the router interface of Vrf20, which orchagent could not remove
func TestPortStateCheckers_RoutedInPreviousVrf(t *testing.T) {
	checker := AsicDb{}.NewChecker(slog.Default(), newTestDB(t, map[string]test.StringMap{
		"COUNTERS_DB": {
			"COUNTERS_PORT_NAME_MAP": test.StringMap{"Ethernet1": "oid:0x10000000003c7"},
			"COUNTERS_RIF_NAME_MAP":  test.StringMap{"Ethernet1": "oid:0x6000000000544"},
		},
		"ASIC_DB": {
			"ASIC_STATE": test.StringMap{
				"SAI_OBJECT_TYPE_ROUTER_INTERFACE": test.StringMap{
					"oid": test.StringMap{
						"0x6000000000544": test.StringMap{
							"SAI_ROUTER_INTERFACE_ATTR_PORT_ID":           "oid:0x10000000003c7",
							"SAI_ROUTER_INTERFACE_ATTR_VIRTUAL_ROUTER_ID": "oid:0x300000000050d",
						},
					},
				},
			},
		},
		"STATE_DB": {
			"INTERFACE_TABLE": test.StringMap{
				"Ethernet1": test.StringMap{"mac_addr": "50:7c:6f:3e:8d:90", "vrf": "Vrf20"},
			},
		},
	}))
	require.NoError(t, checker.Refresh(t.Context()))

	require.Equal(t, []string{
		`STATE_DB INTERFACE_TABLE|Ethernet1 in Vrf46 (in "Vrf20")`,
	}, pending(t, checker.RoutedIn("Ethernet1", "Vrf46")))
	require.Contains(t, pending(t, checker.RoutedIn("Ethernet2", "Vrf46")), "STATE_DB INTERFACE_TABLE|Ethernet2 in Vrf46 (absent)", "an interface intfmgrd did not set up is in no vrf")
}
