package test

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestLoadData(t *testing.T) {
	tests := []struct {
		name string
		data StringMap
		want HashMap
	}{
		{
			name: "empty stringMap",
			data: StringMap{},
			want: nil,
		},
		{
			name: "add empty fields and values to key",
			data: StringMap{
				"LOOPBACK_INTERFACE": StringMap{
					"Loopback0": StringMap{},
				},
			},
			want: HashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					null: null,
				},
			},
		},
		{
			name: "add multiple field-value pairs to multiple keys",
			data: StringMap{
				"PORT": StringMap{
					"Ethernet0": StringMap{
						"admin_status": "up",
						"mtu":          "9000",
					},
					"Ethernet1": StringMap{
						"speed": "25000",
						"alias": "Eth1/2",
					},
				},
			},
			want: HashMap{
				"PORT|Ethernet0": {
					"admin_status": "up",
					"mtu":          "9000",
				},
				"PORT|Ethernet1": {
					"speed": "25000",
					"alias": "Eth1/2",
				},
			},
		},
		{
			name: "add map with one level of nesting and string values",
			data: StringMap{
				"COUNTERS_PORT_NAME_MAP": StringMap{
					"Ethernet0": "oid|0x1000000000020",
					"Ethernet1": "oid|0x1000000000021",
					"Ethernet2": "oid|0x1000000000022",
					"Ethernet3": "",
				},
			},
			want: HashMap{
				"COUNTERS_PORT_NAME_MAP": {
					"Ethernet0": "oid|0x1000000000020",
					"Ethernet1": "oid|0x1000000000021",
					"Ethernet2": "oid|0x1000000000022",
					"Ethernet3": "",
				},
			},
		},
		{
			name: "map with nested keys and null values",
			data: StringMap{
				"VLAN_INTERFACE": StringMap{
					"Vlan1001": StringMap{
						"vrf_name": "Vrf50",
					},
					"Vlan4000":               StringMap{},
					"Vlan4000|10.255.0.1/24": StringMap{},
				},
			},
			want: HashMap{
				"VLAN_INTERFACE|Vlan1001": map[string]string{
					"vrf_name": "Vrf50",
				},
				"VLAN_INTERFACE|Vlan4000|10.255.0.1/24": map[string]string{
					null: null,
				},
				"VLAN_INTERFACE|Vlan4000": map[string]string{
					null: null,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				vc  = StartValkey(t)
				sep = "|"
			)
			defer vc.Close()

			err := LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)

			for k, m := range tt.want {
				cmd := vc.B().Hgetall().Key(k).Build()
				res := vc.Do(ctx, cmd)
				require.NoError(t, res.Error())
				got, err := res.AsStrMap()
				require.NoError(t, err)

				if diff := cmp.Diff(m, got); diff != "" {
					t.Errorf("result for key %s not as expected: %s", k, diff)
				}
			}

			cmd := vc.B().Keys().Pattern("*").Build()
			res := vc.Do(ctx, cmd)
			require.NoError(t, res.Error())
			keys, err := res.AsStrSlice()
			require.NoError(t, err)
			for _, k := range keys {
				if _, found := tt.want[k]; !found {
					t.Errorf("unexpected key in result: %s", k)
				}
			}
		})
	}
}

func TestGetData(t *testing.T) {
	tests := []struct {
		name      string
		separator string
		data      StringMap
		want      HashMap
	}{
		{
			name:      "empty",
			separator: "|",
			want:      HashMap{},
		},
		{
			name:      "get all data",
			separator: "|",
			data: StringMap{
				"LOOPBACK_INTERFACE": StringMap{
					"Loopback0": StringMap{},
				},
				"PORT": StringMap{
					"Ethernet0": StringMap{
						"admin_status": "up",
						"alias":        "Eth1/1",
					},
				},
				"ASIC_STATE": StringMap{
					"SAI_OBJECT_TYPE_BRIDGE_PORT": StringMap{
						"oid": StringMap{
							"0x3a000000001a4a": StringMap{
								"SAI_BRIDGE_PORT_ATTR_ADMIN_STATE": "true",
							},
						},
					},
				},
				"VLAN_INTERFACE": StringMap{
					"Vlan1001": StringMap{
						"vrf_name": "Vrf50",
					},
					"Vlan4000":               StringMap{},
					"Vlan4000|10.255.0.1/24": StringMap{},
				},
			},
			want: HashMap{
				"LOOPBACK_INTERFACE|Loopback0": map[string]string{
					"NULL": "NULL",
				},
				"PORT|Ethernet0": map[string]string{
					"admin_status": "up",
					"alias":        "Eth1/1",
				},
				"ASIC_STATE|SAI_OBJECT_TYPE_BRIDGE_PORT|oid|0x3a000000001a4a": map[string]string{
					"SAI_BRIDGE_PORT_ATTR_ADMIN_STATE": "true",
				},
				"VLAN_INTERFACE|Vlan1001": map[string]string{
					"vrf_name": "Vrf50",
				},
				"VLAN_INTERFACE|Vlan4000": map[string]string{
					"NULL": "NULL",
				},
				"VLAN_INTERFACE|Vlan4000|10.255.0.1/24": map[string]string{
					"NULL": "NULL",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				vc  = StartValkey(t)
			)
			defer vc.Close()

			err := LoadData(ctx, vc, tt.data, tt.separator)
			require.NoError(t, err)

			got, err := GetData(ctx, vc, tt.separator)
			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetData() diff = %s", diff)
			}
		})
	}
}

func Test_getKeysAndValues(t *testing.T) {
	tests := []struct {
		name string
		data StringMap
		want []keysAndValue
	}{
		{
			name: "empty",
			data: StringMap{},
		},
		{
			name: "one level of nesting",
			data: StringMap{
				"PORT": "Ethernet0",
			},
			want: []keysAndValue{
				{
					keys:  []string{"PORT"},
					value: "Ethernet0",
				},
			},
		},
		{
			name: "two levels of nesting with empty value",
			data: StringMap{
				"LOOPBACK_INTERFACE": StringMap{
					"Loopback0": StringMap{},
				},
			},
			want: []keysAndValue{
				{
					keys:  []string{"LOOPBACK_INTERFACE", "Loopback0"},
					value: null,
				},
			},
		},
		{
			name: "two levels of nesting with string value",
			data: StringMap{
				"COUNTERS_PORT_NAME_MAP": StringMap{
					"Ethernet0": "oid:0x1000000000020",
					"Ethernet1": "oid:0x1000000000021",
					"Ethernet2": "oid:0x1000000000022",
					"Ethernet3": "oid:0x1000000000023",
				},
			},
			want: []keysAndValue{
				{
					keys:  []string{"COUNTERS_PORT_NAME_MAP", "Ethernet0"},
					value: "oid:0x1000000000020",
				},
				{
					keys:  []string{"COUNTERS_PORT_NAME_MAP", "Ethernet1"},
					value: "oid:0x1000000000021",
				},
				{
					keys:  []string{"COUNTERS_PORT_NAME_MAP", "Ethernet2"},
					value: "oid:0x1000000000022",
				},
				{
					keys:  []string{"COUNTERS_PORT_NAME_MAP", "Ethernet3"},
					value: "oid:0x1000000000023",
				},
			},
		},
		{
			name: "multiple levels of nesting",
			data: StringMap{
				"PORT": StringMap{
					"Ethernet0": StringMap{
						"admin_status": "up",
						"alias":        "Eth1/1",
					},
				},
			},
			want: []keysAndValue{
				{
					keys:  []string{"PORT", "Ethernet0", "admin_status"},
					value: "up",
				},
				{
					keys:  []string{"PORT", "Ethernet0", "alias"},
					value: "Eth1/1",
				},
			},
		},
		{
			name: "multiple levels of nesting with null values",
			data: StringMap{
				"VLAN_INTERFACE": StringMap{
					"Vlan1001": StringMap{
						"vrf_name": "Vrf50",
					},
					"Vlan4000":               StringMap{},
					"Vlan4000|10.255.0.1/24": StringMap{},
				},
			},
			want: []keysAndValue{
				{
					keys:  []string{"VLAN_INTERFACE", "Vlan1001", "vrf_name"},
					value: "Vrf50",
				},
				{
					keys:  []string{"VLAN_INTERFACE", "Vlan4000"},
					value: null,
				},
				{
					keys:  []string{"VLAN_INTERFACE", "Vlan4000|10.255.0.1/24"},
					value: null,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getKeysAndValues(tt.data)
			slices.SortFunc(got, func(a, b keysAndValue) int {
				return strings.Compare(strings.Join(a.keys, ""), strings.Join(b.keys, ""))
			})
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(keysAndValue{})); diff != "" {
				t.Errorf("flatKeys() diff = %s", diff)
			}
		})
	}
}

func Test_getHashMap(t *testing.T) {
	var (
		separator = "|"
	)

	tests := []struct {
		name string
		kvs  []keysAndValue
		want HashMap
	}{
		{
			name: "empty",
			kvs:  []keysAndValue{},
			want: HashMap{},
		},
		{
			name: "multiple keys with different nesting levels",
			kvs: []keysAndValue{
				{
					keys:  []string{"PORT", "Ethernet0", "admin_status"},
					value: "up",
				},
				{
					keys:  []string{"PORT", "Ethernet0", "alias"},
					value: "Eth1/1",
				},
				{
					keys:  []string{"LOOPBACK_INTERFACE", "Loopback0"},
					value: null,
				},
				{
					keys:  []string{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", "oid", "0x3a000000001a4a", "SAI_BRIDGE_PORT_ATTR_ADMIN_STATE"},
					value: "true",
				},
				{
					keys:  []string{"COUNTERS_PORT_NAME_MAP", "Ethernet0"},
					value: "oid:0x1000000000020",
				},
				{
					keys:  []string{"COUNTERS_PORT_NAME_MAP", "Ethernet1"},
					value: "oid:0x1000000000021",
				},
				{
					keys:  []string{"VLAN_INTERFACE", "Vlan1001", "vrf_name"},
					value: "Vrf50",
				},
				{
					keys:  []string{"VLAN_INTERFACE", "Vlan4000"},
					value: null,
				},
				{
					keys:  []string{"VLAN_INTERFACE", "Vlan4000|10.255.0.1/24"},
					value: null,
				},
			},
			want: HashMap{
				"PORT|Ethernet0": {
					"admin_status": "up",
					"alias":        "Eth1/1",
				},
				"LOOPBACK_INTERFACE|Loopback0": {},
				"ASIC_STATE|SAI_OBJECT_TYPE_BRIDGE_PORT|oid|0x3a000000001a4a": {
					"SAI_BRIDGE_PORT_ATTR_ADMIN_STATE": "true",
				},
				"COUNTERS_PORT_NAME_MAP": {
					"Ethernet0": "oid:0x1000000000020",
					"Ethernet1": "oid:0x1000000000021",
				},
				"VLAN_INTERFACE|Vlan1001": {
					"vrf_name": "Vrf50",
				},
				"VLAN_INTERFACE|Vlan4000":               {},
				"VLAN_INTERFACE|Vlan4000|10.255.0.1/24": {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getHashMap(tt.kvs, separator)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getHashMap() diff = %s", diff)
			}
		})
	}
}

func Test_cutPrefixFromHashMap(t *testing.T) {
	tests := []struct {
		name   string
		hm     HashMap
		prefix string
		want   HashMap
	}{
		{
			name: "empty prefix",
			hm: HashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					null: null,
				},
			},
			prefix: "",
			want: HashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					null: null,
				},
			},
		},
		{
			name: "prefix not found",
			hm: HashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					null: null,
				},
			},
			prefix: "PORT|",
			want:   HashMap{},
		},
		{
			name: "trim prefix where possible and remove the rest",
			hm: HashMap{
				"PORT|Ethernet0": {
					"admin_status": "up",
				},
				"ASIC_STATE|SAI_OBJECT_TYPE_BRIDGE_PORT|oid|0x3a000000001a4a": {
					"SAI_BRIDGE_PORT_ATTR_ADMIN_STATE": "true",
				},
			},
			prefix: "PORT|",
			want: HashMap{
				"Ethernet0": {
					"admin_status": "up",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cutPrefixFromHashMap(tt.hm, tt.prefix)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("cutPrefixFromHashMap() diff = %s", diff)
			}
		})
	}
}
