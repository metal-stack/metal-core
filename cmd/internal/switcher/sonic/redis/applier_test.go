package redis

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// Test_vrfEntries pins the entries to the ones the former CreateVrf, CreateVlan, SuppressNeighbors,
// CreateVlanInterface and CreateVxlanTunnelMap calls wrote, in the same order.
func Test_vrfEntries(t *testing.T) {
	vrf := &types.Vrf{VNI: 104001, VLANID: 1001}

	want := []db.Entry{
		{Key: db.Key{"VRF", "Vrf104001"}, Val: db.Val{"fallback": "false", "vni": "104001"}},
		{Key: db.Key{"VLAN", "Vlan1001"}, Val: db.Val{"vlanid": "1001"}},
		{Key: db.Key{"SUPPRESS_VLAN_NEIGH", "Vlan1001"}, Val: db.Val{"suppress": "on"}},
		{Key: db.Key{"VLAN_INTERFACE", "Vlan1001"}, Val: db.Val{"vrf_name": "Vrf104001"}},
		{Key: db.Key{"VXLAN_TUNNEL_MAP", "vtep", "map_104001_Vlan1001"}, Val: db.Val{"vlan": "Vlan1001", "vni": "104001"}},
	}

	got := vrfEntries("Vrf104001", vrf, "vtep")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("vrfEntries() diff = %s", diff)
	}
}
