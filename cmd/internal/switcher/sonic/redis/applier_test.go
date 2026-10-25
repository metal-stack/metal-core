package redis

import (
	"log/slog"
	"net/netip"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/stretchr/testify/require"
)

func TestApplier_BootPrefixChangeTriggersReconciliation(t *testing.T) {
	mr := miniredis.RunT(t)
	d, err := db.New(&db.Config{
		Databases: map[string]db.Database{
			"APPL_DB":     {Id: 0, Instance: "redis", Separator: ":"},
			"ASIC_DB":     {Id: 1, Instance: "redis", Separator: ":"},
			"CONFIG_DB":   {Id: 4, Instance: "redis", Separator: "|"},
			"COUNTERS_DB": {Id: 2, Instance: "redis", Separator: ":"},
		},
		Instances: map[string]db.Instance{"redis": {Hostname: mr.Addr()}},
	}, db.SchemeRedis)
	require.NoError(t, err)
	a := NewApplier(slog.Default(), d)
	mr.Select(4)
	mr.HSet("PORT|Ethernet0", "admin_status", "up", "mtu", "1500")
	mr.HSet("VLAN|Vlan4000", "vlanid", "4000")
	mr.HSet("VLAN_MEMBER|Vlan4000|Ethernet0", "tagging_mode", "untagged")
	cfg := &types.Conf{PXEVlanID: 4000, Ports: types.Ports{
		Unprovisioned: map[string]*types.UnprovisionedPort{"Ethernet0": {Port: "Ethernet0", BootPrefix: netip.MustParsePrefix("fd00:20::/64")}},
		AdminStatus:   map[string]types.PortStatus{"Ethernet0": types.PortStatusUp},
	}}
	require.NoError(t, a.Apply(t.Context(), cfg))

	// Reset the MTU so applying a changed prefix must reconcile it again.
	mr.HSet("PORT|Ethernet0", "mtu", "1500")

	// A changed prefix must trigger comparison without inspecting netip's private fields.
	changed := *cfg
	changed.Ports.Unprovisioned = map[string]*types.UnprovisionedPort{
		"Ethernet0": {Port: "Ethernet0", BootPrefix: netip.MustParsePrefix("fd00:30::/64")},
	}
	require.NoError(t, a.Apply(t.Context(), &changed))
	require.Equal(t, "9000", mr.HGet("PORT|Ethernet0", "mtu"))
}
