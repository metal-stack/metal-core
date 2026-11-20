package redis

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type testSwitch struct {
	applier  *Applier
	applDB   valkey.Client
	configDB valkey.Client
}

// newTestSwitch runs an applier with the test quirks against the CONFIG_DB and APPL_DB data.
func newTestSwitch(t *testing.T, configDB, applDB test.StringMap) *testSwitch {
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

	client := func(name string) valkey.Client {
		vc, err := valkey.NewClient(valkey.ClientOption{
			InitAddress:  []string{mr.Addr()},
			DisableCache: true,
			SelectDB:     cfg.Databases[name].Id,
		})
		require.NoError(t, err)
		t.Cleanup(vc.Close)
		return vc
	}

	s := &testSwitch{
		applDB:   client("APPL_DB"),
		configDB: client("CONFIG_DB"),
	}
	require.NoError(t, test.LoadData(t.Context(), s.configDB, configDB, "|"))
	require.NoError(t, test.LoadData(t.Context(), s.applDB, applDB, ":"))

	sonicDb, err := db.New(cfg, slog.Default(), db.SchemeRedis)
	require.NoError(t, err)
	s.applier, err = NewApplier(slog.Default(), sonicDb, quirks.NewTestQuirks())
	require.NoError(t, err)
	s.applier.waitTimeout = 300 * time.Millisecond

	return s
}

func (s *testSwitch) configData(t *testing.T) test.HashMap {
	t.Helper()
	data, err := test.GetData(t.Context(), s.configDB, "|")
	require.NoError(t, err)
	return data
}

func (s *testSwitch) del(t *testing.T, vc valkey.Client, key string) {
	t.Helper()
	require.NoError(t, vc.Do(t.Context(), vc.B().Del().Key(key).Build()).Error())
}

// the routes via the machine that kept the router interface of leaf03 Ethernet1 referenced
var routesViaEthernet1 = test.StringMap{
	"ROUTE_TABLE": test.StringMap{
		"Vrf20": test.StringMap{
			"10.128.0.2": test.StringMap{
				"ifname":   "Ethernet1",
				"nexthop":  "fe80::527c:6fff:fe3e:8d90",
				"protocol": "0xba",
			},
		},
	},
}

func TestApplier_configureUnprovisionedPort_routesViaPort(t *testing.T) {
	s := newTestSwitch(t, test.StringMap{
		"PORT": test.StringMap{
			"Ethernet1": test.StringMap{"admin_status": "up", "mtu": "9000"},
		},
		"INTERFACE": test.StringMap{
			"Ethernet1": test.StringMap{"vrf_name": "Vrf20", "ipv6_use_link_local_only": "enable"},
		},
	}, routesViaEthernet1)

	err := s.applier.configureUnprovisionedPort(t.Context(), "Ethernet1", types.PortStatusUp, "Vlan4000")
	require.ErrorContains(t, err, "ROUTE_TABLE:Vrf20:10.128.0.2")

	data := s.configData(t)
	require.Equal(t, map[string]string{"vrf_name": "Vrf20", "ipv6_use_link_local_only": "enable"}, data["INTERFACE|Ethernet1"], "interface must stay in its vrf")
	require.Equal(t, "up", data["PORT|Ethernet1"]["admin_status"], "port must be up again")
	require.NotContains(t, data, "VLAN_MEMBER|Vlan4000|Ethernet1")

	s.del(t, s.applDB, "ROUTE_TABLE:Vrf20:10.128.0.2")

	err = s.applier.configureUnprovisionedPort(t.Context(), "Ethernet1", types.PortStatusUp, "Vlan4000")
	require.NoError(t, err)

	data = s.configData(t)
	require.NotContains(t, data, "INTERFACE|Ethernet1")
	require.Equal(t, "up", data["PORT|Ethernet1"]["admin_status"])
	require.Equal(t, map[string]string{"tagging_mode": "untagged"}, data["VLAN_MEMBER|Vlan4000|Ethernet1"])
}

func TestApplier_configureVrfNeighbor_routesViaPort(t *testing.T) {
	s := newTestSwitch(t, test.StringMap{
		"PORT": test.StringMap{
			"Ethernet1": test.StringMap{"admin_status": "up", "mtu": "9000"},
		},
	}, routesViaEthernet1)

	err := s.applier.configureVrfNeighbor(t.Context(), "Ethernet1", "Vrf46", types.PortStatusUp)
	require.ErrorContains(t, err, "ROUTE_TABLE:Vrf20:10.128.0.2")
	require.NotContains(t, s.configData(t), "INTERFACE|Ethernet1")

	s.del(t, s.applDB, "ROUTE_TABLE:Vrf20:10.128.0.2")

	err = s.applier.configureVrfNeighbor(t.Context(), "Ethernet1", "Vrf46", types.PortStatusUp)
	require.NoError(t, err)
	require.Equal(t, "Vrf46", s.configData(t)["INTERFACE|Ethernet1"]["vrf_name"])
}
