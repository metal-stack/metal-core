package db

import (
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

func TestConfigDB_EnsureEntry(t *testing.T) {
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
				data["SUPPRESS_VLAN_NEIGH|Vlan1004"]["suppress"] = "on"
			},
		},
		{
			name:  "create new without fields",
			data:  configDBTestData,
			entry: Entry{Key: Key{"VLAN_INTERFACE", "Vlan2000"}},
			mods: func(data test.HashMap) {
				data["VLAN_INTERFACE|Vlan2000"] = map[string]string{null: null}
			},
		},
		{
			name:  "existing without fields",
			data:  configDBTestData,
			entry: Entry{Key: Key{"VLAN_INTERFACE", "Vlan4000"}},
			mods:  func(test.HashMap) {},
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
			err = d.EnsureEntry(ctx, tt.entry)
			require.NoError(t, err)
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)

			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.EnsureEntry() data differs = %s", diff)
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
				c: c,
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
			slices.Sort(got)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ConfigDB.GetVrfs() diff = %s", diff)
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

func TestConfigDB_GetVTEPName(t *testing.T) {
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
			got, err := d.GetVTEPName(ctx)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.GetVTEPName() = %v, want %v", got, tt.want)
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
				AdminStatus: string(types.PortStatusUp),
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
		status        types.PortStatus
		mods          func(test.HashMap)
		wantErr       bool
	}{
		{
			name:          "set on non-existing",
			data:          configDBTestData,
			interfaceName: "Ethernet2",
			status:        types.PortStatusDown,
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
			status:        types.PortStatusUp,
			mods:          func(data test.HashMap) {},
		},
		{
			name:          "change existing",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			status:        types.PortStatusDown,
			mods: func(data test.HashMap) {
				data["PORT|Ethernet1"]["admin_status"] = "down"
			},
		},
		{
			name:          "invalid input",
			data:          configDBTestData,
			interfaceName: "Ethernet1",
			status:        "unknown",
			wantErr:       true,
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
			err = d.SetAdminStatus(ctx, tt.interfaceName, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigDB.SetAdminStatusUp() err = %v, want err %v", err, tt.wantErr)
			}
			data, err := test.GetData(ctx, vc, sep)
			require.NoError(t, err)
			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("ConfigDB.SetAdminStatusUp() data differs = %s", diff)
			}
		})
	}
}

func TestConfigDB_GetAdminStatus(t *testing.T) {
	data := test.StringMap{
		"PORT": test.StringMap{
			"Ethernet0": test.StringMap{
				"admin_status": "up",
				"alias":        "Eth1/1",
				"mtu":          "9216",
			},
			"Ethernet1": test.StringMap{
				"alias": "Eth1/2",
				"mtu":   "9000",
			},
			"Ethernet2": test.StringMap{
				"admin_status": "down",
				"alias":        "Eth1/3",
				"mtu":          "9000",
			},
		},
	}

	tests := []struct {
		name          string
		data          test.StringMap
		interfaceName string
		want          types.PortStatus
	}{
		{
			name:          "interface is up",
			data:          data,
			interfaceName: "Ethernet0",
			want:          types.PortStatusUp,
		},
		{
			name:          "interface is down",
			data:          data,
			interfaceName: "Ethernet2",
			want:          types.PortStatusDown,
		},
		{
			name:          "no admin_status defined",
			data:          data,
			interfaceName: "Ethernet1",
			want:          "",
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
			got, err := d.GetAdminStatus(ctx, tt.interfaceName)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ConfigDB.GetAdminStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
