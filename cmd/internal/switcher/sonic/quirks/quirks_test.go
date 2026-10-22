package quirks

import (
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func TestQuirkProfiles(t *testing.T) {
	for name, newQuirks := range QuirkProfiles {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, name, newQuirks().Name())
		})
	}
}

func TestQuirks_VrfEntries(t *testing.T) {
	vrf := &types.Vrf{VNI: 104001, VLANID: 1001}

	tests := []struct {
		name   string
		quirks Quirks
		want   []db.Entry
	}{
		{
			name:   "community",
			quirks: communityQuirks{},
			want: []db.Entry{
				{Key: db.Key{"VRF", "Vrf104001"}, Val: db.Val{"fallback": "false", "vni": "104001"}},
				{Key: db.Key{"VLAN", "Vlan1001"}, Val: db.Val{"vlanid": "1001"}},
				{Key: db.Key{"SUPPRESS_VLAN_NEIGH", "Vlan1001"}, Val: db.Val{"suppress": "on"}},
				{Key: db.Key{"VLAN_INTERFACE", "Vlan1001"}, Val: db.Val{"vrf_name": "Vrf104001"}},
				{Key: db.Key{"VXLAN_TUNNEL_MAP", "vtep", "map_104001_Vlan1001"}, Val: db.Val{"vlan": "Vlan1001", "vni": "104001"}},
			},
		},
		{
			name:   "asternos",
			quirks: asterNOSQuirks{},
			want: []db.Entry{
				{Key: db.Key{"VRF", "Vrf104001"}, Val: db.Val{"vni": "104001", "source_vtep": "vtep"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.quirks.VrfEntries("Vrf104001", vrf, "vtep")
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("VrfEntries() diff = %s", diff)
			}
		})
	}
}

func TestQuirks_PortState(t *testing.T) {
	tests := []struct {
		name   string
		quirks Quirks
		want   PortStateChecker
	}{
		{
			name:   "community",
			quirks: communityQuirks{},
			want:   &asicDBPortState{},
		},
		{
			name:   "asternos",
			quirks: asterNOSQuirks{},
			want:   &stateDBPortState{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.IsType(t, tt.want, tt.quirks.PortState(slog.Default(), &db.DB{}))
		})
	}
}
