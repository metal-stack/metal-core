package db

import (
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

func TestClient_Dump(t *testing.T) {
	var (
		ctx = t.Context()
		sep = "|"
		vc  = test.StartValkey(t)
	)
	defer vc.Close()

	err := test.LoadData(ctx, vc, test.StringMap{
		"VLAN": test.StringMap{
			"Vlan4000": test.StringMap{"vlanid": "4000"},
		},
		"VLAN_MEMBER": test.StringMap{
			"Vlan4000": test.StringMap{
				"Ethernet0": test.StringMap{"tagging_mode": "untagged"},
			},
		},
		"VRF": test.StringMap{
			"Vrf46": test.StringMap{},
		},
		"PORT": test.StringMap{
			"Ethernet0": test.StringMap{"admin_status": "up"},
		},
	}, sep)
	require.NoError(t, err)

	c := &Client{rdb: vc, sep: sep}
	dump, err := c.Dump(ctx, Key{"VLAN"}, Key{"VLAN_MEMBER"}, Key{"VRF"})
	require.NoError(t, err)

	val, ok := dump.Get(Key{"VLAN_MEMBER", "Vlan4000", "Ethernet0"})
	require.True(t, ok)
	require.Equal(t, Val{"tagging_mode": "untagged"}, val)

	val, ok = dump.Get(Key{"VRF", "Vrf46"})
	require.True(t, ok)
	require.Empty(t, val, "the NULL placeholder is no field")

	require.False(t, dump.Has(Key{"PORT", "Ethernet0"}), "tables that were not dumped are empty")
	require.Equal(t, []Key{{"VLAN", "Vlan4000"}}, dump.Keys(Key{"VLAN"}), "VLAN_MEMBER entries are not part of the VLAN table")
	require.Equal(t, []Key{{"VLAN_MEMBER", "Vlan4000", "Ethernet0"}}, dump.Keys(Key{"VLAN_MEMBER"}))
	require.Equal(t, []string{"Vlan4000|Ethernet0"}, dump.Items(Key{"VLAN_MEMBER"}))
	require.Equal(t, "VLAN_MEMBER|Vlan4000|Ethernet0", dump.String(Key{"VLAN_MEMBER", "Vlan4000", "Ethernet0"}))
}
