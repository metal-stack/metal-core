package db

import (
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/stretchr/testify/require"
)

var (
	configDBTestData = test.StringMap{
		"INTERFACE": test.StringMap{
			"Ethernet0": test.StringMap{
				"ipv6_use_link_local_only": "enable",
				"vrf_name":                 "Vrf102",
			},
			"Ethernet1": test.StringMap{
				"ipv6_use_link_local_only": "enable",
			},
			"Ethernet3": test.StringMap{
				"ipv6_use_link_local_only": "disable",
			},
		},
		"LOOPBACK_INTERFACE": test.StringMap{
			"Loopback0": test.StringMap{},
		},
		"PORT": test.StringMap{
			"Ethernet0": test.StringMap{
				"admin_status": "up",
				"alias":        "Eth1/1",
				"mtu":          "9216",
			},
			"Ethernet1": test.StringMap{
				"admin_status": "up",
				"alias":        "Eth1/2",
				"mtu":          "9000",
			},
		},
		"SUPPRESS_VLAN_NEIGH": test.StringMap{
			"Vlan1001": test.StringMap{
				"suppress": "on",
			},
			"Vlan1002": test.StringMap{
				"suppress": "on",
			},
			"Vlan1003": test.StringMap{
				"suppress": "off",
			},
			"Vlan1004": test.StringMap{},
		},
		"VLAN": test.StringMap{
			"Vlan1001": test.StringMap{
				"vlanid": "1001",
			},
			"Vlan4000": test.StringMap{
				"vlanid": "4000",
			},
		},
		"VLAN_INTERFACE": test.StringMap{
			"Vlan1001": test.StringMap{
				"vrf_name": "Vrf50",
			},
			"Vlan4000":               test.StringMap{},
			"Vlan4000|10.255.0.1/24": test.StringMap{},
		},
		"VLAN_MEMBER": test.StringMap{
			"Vlan4000|Ethernet0": test.StringMap{
				"tagging_mode": "untagged",
			},
			"Vlan4000|Ethernet1": test.StringMap{
				"tagging_mode": "untagged",
			},
		},
		"VRF": test.StringMap{
			"Vrf102": test.StringMap{
				"fallback": "false",
				"vni":      "102",
			},
			"Vrf110": test.StringMap{
				"fallback": "false",
				"vni":      "110",
			},
		},
		"VXLAN_EVPN_NVO": test.StringMap{
			"nvo": test.StringMap{
				"source_vtep": "vtep",
			},
		},
		"VXLAN_TUNNEL": test.StringMap{
			"vtep": test.StringMap{
				"src_ip": "10.0.7.7",
			},
		},
		"VXLAN_TUNNEL_MAP": test.StringMap{
			"vtep|map_102_Vlan1005": test.StringMap{
				"vlan": "Vlan1005",
				"vni":  "102",
			},
		},
	}
)

func TestConfigDB_MergeEntry(t *testing.T) {
	tests := []struct {
		name  string
		data  test.StringMap
		entry Entry
		mods  func(test.HashMap)
	}{
		{
			name:  "create new",
			data:  configDBTestData,
			entry: Entry{Key: Key{"VRF", "Vrf200"}, Val: Val{"fallback": "false", "vni": "200"}},
			mods: func(data test.HashMap) {
				data["VRF|Vrf200"] = map[string]string{
					"fallback": "false",
					"vni":      "200",
				}
			},
		},
		{
			name:  "existing with same fields",
			data:  configDBTestData,
			entry: Entry{Key: Key{"VRF", "Vrf102"}, Val: Val{"fallback": "false", "vni": "102"}},
			mods:  func(test.HashMap) {},
		},
		{
			name:  "existing with subset of fields",
			data:  configDBTestData,
			entry: Entry{Key: Key{"VRF", "Vrf102"}, Val: Val{"vni": "102"}},
			mods:  func(test.HashMap) {},
		},
		{
			name:  "existing with different field",
			data:  configDBTestData,
			entry: Entry{Key: Key{"VRF", "Vrf102"}, Val: Val{"vni": "103"}},
			mods: func(data test.HashMap) {
				data["VRF|Vrf102"]["vni"] = "103"
			},
		},
		{
			name:  "existing without desired field",
			data:  configDBTestData,
			entry: Entry{Key: Key{"SUPPRESS_VLAN_NEIGH", "Vlan1004"}, Val: Val{"suppress": "on"}},
			mods: func(data test.HashMap) {
				data["SUPPRESS_VLAN_NEIGH|Vlan1004"] = map[string]string{"suppress": "on"}
			},
		},
		{
			name:  "empty value is a no-op",
			data:  configDBTestData,
			entry: Entry{Key: Key{"VLAN_INTERFACE", "Vlan2000"}},
			mods:  func(test.HashMap) {},
		},
		{
			name:  "replace null placeholder with real fields",
			data:  configDBTestData,
			entry: Entry{Key: Key{"VLAN_INTERFACE", "Vlan4000"}, Val: Val{"vrf_name": "Vrf4000"}},
			mods: func(data test.HashMap) {
				data["VLAN_INTERFACE|Vlan4000"] = map[string]string{"vrf_name": "Vrf4000"}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = "|"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)
			initData, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			if tt.mods != nil {
				tt.mods(initData)
			}

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &ConfigDB{
				log: slog.Default(),
				c:   c,
			}
			err = d.MergeEntry(ctx, tt.entry)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.MergeEntry() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_DeleteEntry(t *testing.T) {
	tests := []struct {
		name        string
		data        test.StringMap
		key         Key
		wantDeleted bool
		mods        func(test.HashMap)
	}{
		{
			name:        "delete non-existing",
			data:        configDBTestData,
			key:         Key{"VRF", "Vrf200"},
			wantDeleted: false,
			mods:        func(test.HashMap) {},
		},
		{
			name:        "delete existing",
			data:        configDBTestData,
			key:         Key{"VXLAN_TUNNEL_MAP", "vtep", "map_102_Vlan1005"},
			wantDeleted: true,
			mods: func(data test.HashMap) {
				delete(data, "VXLAN_TUNNEL_MAP|vtep|map_102_Vlan1005")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = "|"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)
			initData, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			if tt.mods != nil {
				tt.mods(initData)
			}

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &ConfigDB{
				log: slog.Default(),
				c:   c,
			}
			deleted, err := d.DeleteEntry(ctx, tt.key)
			require.NoError(t, err)
			require.Equal(t, tt.wantDeleted, deleted)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.DeleteEntry() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_GetPorts(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		want []*Port
	}{
		{
			name: "get all ports",
			data: configDBTestData,
			want: []*Port{
				{
					Name:        "Ethernet0",
					Alias:       "Eth1/1",
					AdminStatus: string(types.PortStatusUp),
					Mtu:         "9216",
				},
				{
					Name:        "Ethernet1",
					Alias:       "Eth1/2",
					AdminStatus: string(types.PortStatusUp),
					Mtu:         "9000",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = "|"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &ConfigDB{
				log: slog.Default(),
				c:   c,
			}
			got, err := d.GetPorts(ctx)
			require.NoError(t, err)
			slices.SortFunc(got, func(a, b *Port) int {
				return strings.Compare(a.Name, b.Name)
			})
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ConfigDB.GetPorts() diff = %s", diff)
			}
		})
	}
}

func TestConfigDB_SetFields(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		key  Key
		set  Val
		del  []string
		want map[string]string
	}{
		{
			name: "new entry",
			key:  Key{"VRF", "Vrf46"},
			set:  Val{"vni": "46"},
			want: map[string]string{"vni": "46"},
		},
		{
			name: "new entry without fields",
			key:  Key{"VRF", "Vrf46"},
			want: map[string]string{"NULL": "NULL"},
		},
		{
			name: "fields replace the NULL placeholder",
			data: test.StringMap{"VRF": test.StringMap{"Vrf46": test.StringMap{}}},
			key:  Key{"VRF", "Vrf46"},
			set:  Val{"vni": "46"},
			want: map[string]string{"vni": "46"},
		},
		{
			name: "set and delete fields, others are kept",
			data: test.StringMap{"PORT": test.StringMap{"Ethernet0": test.StringMap{"alias": "Eth1/1", "mtu": "9216", "fec": "rs"}}},
			key:  Key{"PORT", "Ethernet0"},
			set:  Val{"mtu": "9000"},
			del:  []string{"fec"},
			want: map[string]string{"alias": "Eth1/1", "mtu": "9000"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = "|"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			require.NoError(t, test.LoadData(ctx, vc, tt.data, sep))
			d := &ConfigDB{log: slog.Default(), c: &Client{rdb: vc, sep: sep}}

			require.NoError(t, d.SetFields(ctx, tt.key, tt.set, tt.del))

			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			require.Equal(t, tt.want, data[tt.key.toString(sep)])
		})
	}
}
