package db

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
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

func TestConfigDB_ExistVlan(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		want bool
	}{
		{
			name: "not existing",
			data: configDBTestData,
			vid:  2000,
			want: false,
		},
		{
			name: "existing",
			data: configDBTestData,
			vid:  4000,
			want: true,
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
				c: c,
			}
			got, err := d.ExistVlan(ctx, tt.vid)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.ExistVlan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDB_ExistVlanInterface(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		want bool
	}{
		{
			name: "not existing",
			data: configDBTestData,
			vid:  2000,
			want: false,
		},
		{
			name: "existing",
			data: configDBTestData,
			vid:  1001,
			want: true,
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
				c: c,
			}
			got, err := d.ExistVlanInterface(ctx, tt.vid)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.ExistVlanInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDB_CreateVlan(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		mods func(test.HashMap)
		vid  uint16
	}{
		{
			name: "create existing",
			data: configDBTestData,
			vid:  4000,
		},
		{
			name: "create new",
			data: configDBTestData,
			mods: func(data test.HashMap) {
				data["VLAN|Vlan2000"] = map[string]string{
					"vlanid": "2000",
				}
			},
			vid: 2000,
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
				c: c,
			}
			err = d.CreateVlan(ctx, tt.vid)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.CreateVlan() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_CreateVlanInterface(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		vrf  string
		mods func(test.HashMap)
	}{
		{
			name: "create existing",
			data: configDBTestData,
			vid:  1001,
			vrf:  "Vrf50",
			mods: func(test.HashMap) {},
		},
		{
			name: "change existing",
			data: configDBTestData,
			vid:  1001,
			vrf:  "Vrf40",
			mods: func(data test.HashMap) {
				data["VLAN_INTERFACE|Vlan1001"]["vrf_name"] = "Vrf40"
			},
		},
		{
			name: "empty vrf",
			data: configDBTestData,
			vid:  1001,
			vrf:  "",
			mods: func(data test.HashMap) {
				data["VLAN_INTERFACE|Vlan1001"]["vrf_name"] = ""
			},
		},
		{
			name: "new",
			data: configDBTestData,
			vid:  2000,
			vrf:  "Vrf100",
			mods: func(data test.HashMap) {
				data["VLAN_INTERFACE|Vlan2000"] = map[string]string{
					"vrf_name": "Vrf100",
				}
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
				c: c,
			}
			err = d.CreateVlanInterface(ctx, tt.vid, tt.vrf)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.CreateVlan() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_DeleteVlan(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		mods func(test.HashMap)
	}{
		{
			name: "delete non-existing",
			data: configDBTestData,
			vid:  3000,
			mods: func(test.HashMap) {},
		},
		{
			name: "delete existing",
			data: configDBTestData,
			vid:  4000,
			mods: func(data test.HashMap) {
				delete(data, "VLAN|Vlan4000")
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
				c: c,
			}
			err = d.DeleteVlan(ctx, tt.vid)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.DeleteVlan() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_DeleteVlanInterface(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		mods func(test.HashMap)
	}{
		{
			name: "delete non-existing",
			data: configDBTestData,
			vid:  3000,
			mods: func(test.HashMap) {},
		},
		{
			name: "delete existing",
			data: configDBTestData,
			vid:  1001,
			mods: func(data test.HashMap) {
				delete(data, "VLAN_INTERFACE|Vlan1001")
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
				c: c,
			}
			err = d.DeleteVlanInterface(ctx, tt.vid)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.DeleteVlanInterface() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_DeleteVlanMember(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		vlan          string
		mods          func(test.HashMap)
	}{
		{
			name:          "delete non-existing member of existing vlan",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			vlan:          "Vlan4000",
			mods:          func(test.HashMap) {},
		},
		{
			name:          "delete from non-existing vlan",
			data:          configDBTestData,
			interfaceName: "Ethernet0",
			vlan:          "Vlan2000",
			mods:          func(test.HashMap) {},
		},
		{
			name:          "delete exisiting member",
			data:          configDBTestData,
			interfaceName: "Ethernet0",
			vlan:          "Vlan4000",
			mods: func(data test.HashMap) {
				delete(data, "VLAN_MEMBER|Vlan4000|Ethernet0")
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
				c: c,
			}
			err = d.DeleteVlanMember(ctx, tt.interfaceName, tt.vlan)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.DeleteVlanMember() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_AreNeighborsSuppressed(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		want bool
	}{
		{
			name: "non-existing vlan",
			data: configDBTestData,
			vid:  3000,
			want: false,
		},
		{
			name: "vlan neighbor suppression off",
			data: configDBTestData,
			vid:  1003,
			want: false,
		},
		{
			name: "vlan neighbor suppression field does not exist",
			data: configDBTestData,
			vid:  1004,
			want: false,
		},
		{
			name: "vlan neighbor suppression on",
			data: configDBTestData,
			vid:  1001,
			want: true,
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
				c: c,
			}
			got, err := d.AreNeighborsSuppressed(ctx, tt.vid)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.AreNeighborsSuppressed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDB_SuppressNeighbors(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		mods func(test.HashMap)
	}{
		{
			name: "suppress existing suppressed",
			data: configDBTestData,
			vid:  1001,
		},
		{
			name: "suppress existing not suppressed",
			data: configDBTestData,
			vid:  1003,
			mods: func(data test.HashMap) {
				data["SUPPRESS_VLAN_NEIGH|Vlan1003"]["suppress"] = "on"
			},
		},
		{
			name: "suppress existing with not suppression set",
			data: configDBTestData,
			vid:  1004,
			mods: func(data test.HashMap) {
				data["SUPPRESS_VLAN_NEIGH|Vlan1004"]["suppress"] = "on"
			},
		},
		{
			name: "suppress existing vlan that does not exist in suppression map",
			data: configDBTestData,
			vid:  4000,
			mods: func(data test.HashMap) {
				data["SUPPRESS_VLAN_NEIGH|Vlan4000"] = map[string]string{
					"suppress": "on",
				}
			},
		},
		{
			name: "suppress non-existing vlan",
			data: configDBTestData,
			vid:  2000,
			mods: func(data test.HashMap) {
				data["SUPPRESS_VLAN_NEIGH|Vlan2000"] = map[string]string{
					"suppress": "on",
				}
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
				c: c,
			}
			err = d.SuppressNeighbors(ctx, tt.vid)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.SuppressNeighbors() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_DeleteNeighborSuppression(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		mods func(test.HashMap)
	}{
		{
			name: "delete non-existing",
			data: configDBTestData,
			vid:  4000,
			mods: func(test.HashMap) {},
		},
		{
			name: "delete existing",
			data: configDBTestData,
			vid:  1001,
			mods: func(data test.HashMap) {
				delete(data, "SUPPRESS_VLAN_NEIGH|Vlan1001")
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
				c: c,
			}
			err = d.DeleteNeighborSuppression(ctx, tt.vid)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.DeleteNeighborSuppression() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_GetVlanMembership(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		want          []string
	}{
		{
			name:          "non-existing",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			want:          []string{},
		},
		{
			name:          "existing",
			data:          configDBTestData,
			interfaceName: "Ethernet0",
			want:          []string{"Vlan4000"},
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
				c: c,
			}
			got, err := d.GetVlanMembership(ctx, tt.interfaceName)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ConfigDB.GetVlanMembership() diff = %s", diff)
			}
		})
	}
}

func TestConfigDB_SetVlanMember(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		vlan          string
		mods          func(test.HashMap)
	}{
		{
			name:          "add existing membership",
			data:          configDBTestData,
			interfaceName: "Ethernet0",
			vlan:          "Vlan4000",
			mods:          func(test.HashMap) {},
		},
		{
			name:          "add new interface to vlan that already has members",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			vlan:          "Vlan4000",
			mods: func(data test.HashMap) {
				data["VLAN_MEMBER|Vlan4000|Ethernet2"] = map[string]string{
					"tagging_mode": "untagged",
				}
			},
		},
		{
			name:          "add new interface to vlan with no members yet",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			vlan:          "Vlan1001",
			mods: func(data test.HashMap) {
				data["VLAN_MEMBER|Vlan1001|Ethernet2"] = map[string]string{
					"tagging_mode": "untagged",
				}
			},
		},
		{
			name:          "add member to non-existing vlan",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			vlan:          "Vlan2000",
			mods: func(data test.HashMap) {
				data["VLAN_MEMBER|Vlan2000|Ethernet2"] = map[string]string{
					"tagging_mode": "untagged",
				}
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
				c: c,
			}
			err = d.SetVlanMember(ctx, tt.interfaceName, tt.vlan)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.SetVlanMember() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_GetVrfs(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		want []string
	}{
		{
			name: "get all vrfs",
			data: configDBTestData,
			want: []string{"Vrf102", "Vrf110"},
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
				c: c,
			}
			got, err := d.GetVrfs(ctx)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ConfigDB.GetVrfs() diff = %s", diff)
			}
		})
	}
}

func TestConfigDB_ExistVrf(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vrf  string
		want bool
	}{
		{
			name: "exists",
			data: configDBTestData,
			vrf:  "Vrf102",
			want: true,
		},
		{
			name: "not exists",
			data: configDBTestData,
			vrf:  "Vrf122",
			want: false,
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
				c: c,
			}
			got, err := d.ExistVrf(ctx, tt.vrf)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.ExistVrf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDB_CreateVrf(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vrf  string
		vni  uint32
		mods func(test.HashMap)
	}{
		{
			name: "create existing",
			data: configDBTestData,
			vrf:  "Vrf102",
			vni:  102,
			mods: func(test.HashMap) {},
		},
		{
			name: "create existing with different vni",
			data: configDBTestData,
			vrf:  "Vrf102",
			vni:  103,
			mods: func(data test.HashMap) {
				data["VRF|Vrf102"]["vni"] = "103"
			},
		},
		{
			name: "create new",
			data: configDBTestData,
			vrf:  "Vrf200",
			vni:  200,
			mods: func(data test.HashMap) {
				data["VRF|Vrf200"] = map[string]string{
					"fallback": "false",
					"vni":      "200",
				}
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
				c: c,
			}
			err = d.CreateVrf(ctx, tt.vrf, tt.vni)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.CreateVrf() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_DeleteVrf(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vrf  string
		mods func(test.HashMap)
	}{
		{
			name: "delete non-existing",
			data: configDBTestData,
			vrf:  "Vrf200",
			mods: func(test.HashMap) {},
		},
		{
			name: "delete existing",
			data: configDBTestData,
			vrf:  "Vrf102",
			mods: func(data test.HashMap) {
				delete(data, "VRF|Vrf102")
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
				c: c,
			}
			err = d.DeleteVrf(ctx, tt.vrf)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.DeleteVrf() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_SetVrfMember(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		vrf           string
		mods          func(test.HashMap)
	}{
		{
			name:          "set existing membership",
			data:          configDBTestData,
			interfaceName: "Ethernet0",
			vrf:           "Vrf102",
		},
		{
			name:          "set vrf member on new interface",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			vrf:           "Vrf110",
			mods: func(data test.HashMap) {
				data["INTERFACE|Ethernet2"] = map[string]string{
					"vrf_name":                 "Vrf110",
					"ipv6_use_link_local_only": "enable",
				}
			},
		},
		{
			name:          "set member for non-existing vrf",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			vrf:           "Vrf200",
			mods: func(data test.HashMap) {
				data["INTERFACE|Ethernet2"] = map[string]string{
					"vrf_name":                 "Vrf200",
					"ipv6_use_link_local_only": "enable",
				}
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
				c: c,
			}
			err = d.SetVrfMember(ctx, tt.interfaceName, tt.vrf)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.SetVrfMember() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_GetVrfMembership(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		want          string
	}{
		{
			name:          "interface in default vrf",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			want:          "",
		},
		{
			name:          "interface in vrf",
			data:          configDBTestData,
			interfaceName: "Ethernet0",
			want:          "Vrf102",
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
				c: c,
			}
			got, err := d.GetVrfMembership(ctx, tt.interfaceName)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.GetVrfMembership() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDB_ExistVxlanTunnelMap(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		vni  uint32
		want bool
	}{
		{
			name: "not exists",
			data: configDBTestData,
			vid:  1001,
			vni:  102,
			want: false,
		},
		{
			name: "exists",
			data: configDBTestData,
			vid:  1005,
			vni:  102,
			want: true,
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
				c: c,
			}
			got, err := d.ExistVxlanTunnelMap(ctx, tt.vid, tt.vni)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.ExistVxlanTunnelMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDB_CreateVxlanTunnelMap(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		vni  uint32
		mods func(test.HashMap)
	}{
		{
			name: "create existing",
			data: configDBTestData,
			vid:  1005,
			vni:  102,
			mods: func(test.HashMap) {},
		},
		{
			name: "create new",
			data: configDBTestData,
			vid:  1001,
			vni:  200,
			mods: func(data test.HashMap) {
				data["VXLAN_TUNNEL_MAP|vtep|map_200_Vlan1001"] = map[string]string{
					"vlan": "Vlan1001",
					"vni":  "200",
				}
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
				c: c,
			}
			err = d.CreateVxlanTunnelMap(ctx, tt.vid, tt.vni)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.CreateVxlanTunnelMap() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_DeleteVxlanTunnelMap(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vid  uint16
		vni  uint32
		mods func(test.HashMap)
	}{
		{
			name: "delete non-existing",
			data: configDBTestData,
			vid:  1000,
			vni:  100,
			mods: func(test.HashMap) {},
		},
		{
			name: "delete existing",
			data: configDBTestData,
			vid:  1005,
			vni:  102,
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
				c: c,
			}
			err = d.DeleteVxlanTunnelMap(ctx, tt.vid, tt.vni)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.DeleteVxlanTunnelMap() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_FindVxlanTunnelMapByVni(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		vni  uint32
		want *VxlanMap
	}{
		{
			name: "non-existing",
			data: configDBTestData,
			vni:  100,
			want: nil,
		},
		{
			name: "existing",
			data: configDBTestData,
			vni:  102,
			want: &VxlanMap{
				Vni:  "102",
				Vlan: "Vlan1005",
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
				c: c,
			}
			got, err := d.FindVxlanTunnelMapByVni(ctx, tt.vni)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ConfigDB.FindVxlanTunnelMapByVni() diff = %s", diff)
			}
		})
	}
}

func TestConfigDB_getVTEPName(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		want string
	}{
		{
			name: "get vtep name",
			data: configDBTestData,
			want: "vtep",
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
				c: c,
			}
			got, err := d.getVTEPName(ctx)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.getVTEPName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDB_DeleteInterfaceConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		mods          func(test.HashMap)
	}{
		{
			name:          "delete non-existing",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			mods:          func(test.HashMap) {},
		},
		{
			name:          "delete existing",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			mods: func(data test.HashMap) {
				delete(data, "INTERFACE|Ethernet1")
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
				c: c,
			}
			err = d.DeleteInterfaceConfiguration(ctx, tt.interfaceName)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.DeleteInterfaceConfiguration() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_IsLinkLocalOnly(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		want          bool
	}{
		{
			name:          "not link local only",
			data:          configDBTestData,
			interfaceName: "Ethernet3",
			want:          false,
		},
		{
			name:          "link local only",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			want:          true,
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
				c: c,
			}
			got, err := d.IsLinkLocalOnly(ctx, tt.interfaceName)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.IsLinkLocalOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDB_EnableLinkLocalOnly(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		mods          func(test.HashMap)
	}{
		{
			name:          "enable where already enabled",
			data:          configDBTestData,
			interfaceName: "Ethernet0",
			mods:          func(test.HashMap) {},
		},
		{
			name:          "enable new",
			data:          configDBTestData,
			interfaceName: "Ethernet4",
			mods: func(data test.HashMap) {
				data["INTERFACE|Ethernet4"] = map[string]string{
					"ipv6_use_link_local_only": "enable",
				}
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
				c: c,
			}
			err = d.EnableLinkLocalOnly(ctx, tt.interfaceName)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.EnableLinkLocalOnly() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_GetPort(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		want          *Port
	}{
		{
			name:          "get non-existing",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			want:          nil,
		},
		{
			name:          "get existing",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			want: &Port{
				Name:        "Ethernet1",
				Alias:       "Eth1/2",
				AdminStatus: true,
				Mtu:         "9000",
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
				c: c,
			}
			got, err := d.GetPort(ctx, tt.interfaceName)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ConfigDB.GetPort() diff = %s", diff)
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
					AdminStatus: true,
					Mtu:         "9216",
				},
				{
					Name:        "Ethernet1",
					Alias:       "Eth1/2",
					AdminStatus: true,
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
				c: c,
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

func TestConfigDB_SetPortMtu(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		val           string
		mods          func(test.HashMap)
	}{
		{
			name:          "set mtu of non-existing",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			val:           "9000",
			mods: func(data test.HashMap) {
				data["PORT|Ethernet2"] = map[string]string{
					"mtu": "9000",
				}
			},
		},
		{
			name:          "set mtu of existing",
			data:          configDBTestData,
			interfaceName: "Ethernet0",
			val:           "9000",
			mods: func(data test.HashMap) {
				data["PORT|Ethernet0"]["mtu"] = "9000"
			},
		},
		{
			name:          "set same mtu as existing",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			val:           "9000",
			mods:          func(data test.HashMap) {},
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
				c: c,
			}
			err = d.SetPortMtu(ctx, tt.interfaceName, tt.val)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.SetPortMtu() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_SetAdminStatusUp(t *testing.T) {
	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		up            bool
		mods          func(test.HashMap)
	}{
		{
			name:          "set on non-existing",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			up:            false,
			mods: func(data test.HashMap) {
				data["PORT|Ethernet2"] = map[string]string{
					"admin_status": "down",
				}
			},
		},
		{
			name:          "set same as existing",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			up:            true,
			mods:          func(data test.HashMap) {},
		},
		{
			name:          "change existing",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			up:            false,
			mods: func(data test.HashMap) {
				data["PORT|Ethernet1"]["admin_status"] = "down"
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
				c: c,
			}
			err = d.SetAdminStatusUp(ctx, tt.interfaceName, tt.up)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.SetAdminStatusUp() data differs = %s", diff)
			}
		})
	}
}
