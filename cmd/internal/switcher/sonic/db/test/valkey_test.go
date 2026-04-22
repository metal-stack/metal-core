package test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestLoadData(t *testing.T) {
	tests := []struct {
		name string
		data StringMap
		want hashMap
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
			want: hashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					"NULL": "NULL",
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
			want: hashMap{
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
		want      StringMap
	}{
		{
			name:      "empty",
			separator: "|",
			want:      StringMap{},
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
			},
			want: StringMap{
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
			name: "two levels of nesting",
			data: StringMap{
				"LOOPBACK_INTERFACE": StringMap{
					"Loopback0": StringMap{},
				},
			},
			want: []keysAndValue{
				{
					keys:  []string{"LOOPBACK_INTERFACE", "Loopback0"},
					value: "",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getKeysAndValues(tt.data)
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
		want hashMap
	}{
		{
			name: "empty",
			kvs:  []keysAndValue{},
			want: hashMap{},
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
					value: "",
				},
				{
					keys:  []string{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", "oid", "0x3a000000001a4a", "SAI_BRIDGE_PORT_ATTR_ADMIN_STATE"},
					value: "true",
				},
			},
			want: hashMap{
				"PORT|Ethernet0": {
					"admin_status": "up",
					"alias":        "Eth1/1",
				},
				"LOOPBACK_INTERFACE|Loopback0": {},
				"ASIC_STATE|SAI_OBJECT_TYPE_BRIDGE_PORT|oid|0x3a000000001a4a": {
					"SAI_BRIDGE_PORT_ATTR_ADMIN_STATE": "true",
				},
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

func Test_stringMapFromHashMap(t *testing.T) {
	tests := []struct {
		name      string
		hm        hashMap
		separator string
		want      StringMap
	}{
		{
			name:      "empty",
			hm:        hashMap{},
			separator: "|",
			want:      StringMap{},
		},
		{
			name: "flat map",
			hm: hashMap{
				"PORT": {
					"Ethernet0": "up",
				},
			},
			separator: "|",
			want: StringMap{
				"PORT": StringMap{
					"Ethernet0": "up",
				},
			},
		},
		{
			name: "one level of nesting",
			hm: hashMap{
				"LOOPBACK_INTERFACE|Loopback0": {},
			},
			separator: "|",
			want: StringMap{
				"LOOPBACK_INTERFACE": StringMap{
					"Loopback0": StringMap{},
				},
			},
		},
		{
			name: "handle null entries",
			hm: hashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					"NULL": "NULL",
				},
			},
			separator: "|",
			want: StringMap{
				"LOOPBACK_INTERFACE": StringMap{
					"Loopback0": StringMap{},
				},
			},
		},
		{
			name: "multiple levels of nesting",
			hm: hashMap{
				"PORT|Ethernet0": {
					"admin_status": "up",
					"alias":        "Eth1/1",
				},
				"LOOPBACK_INTERFACE|Loopback0": {
					"NULL": "NULL",
				},
				"ASIC_STATE|SAI_OBJECT_TYPE_BRIDGE_PORT|oid|0x3a000000001a4a": {
					"SAI_BRIDGE_PORT_ATTR_ADMIN_STATE": "true",
				},
			},
			separator: "|",
			want: StringMap{
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringMapFromHashMap(tt.hm, tt.separator)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("stringMapFromHashMap() diff = %s", diff)
			}
		})
	}
}

func Test_cutPrefixFromHashMap(t *testing.T) {
	tests := []struct {
		name   string
		hm     hashMap
		prefix string
		want   hashMap
	}{
		{
			name: "empty prefix",
			hm: hashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					"NULL": "NULL",
				},
			},
			prefix: "",
			want: hashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					"NULL": "NULL",
				},
			},
		},
		{
			name: "prefix not found",
			hm: hashMap{
				"LOOPBACK_INTERFACE|Loopback0": {
					"NULL": "NULL",
				},
			},
			prefix: "PORT|",
			want:   hashMap{},
		},
		{
			name: "trim prefix where possible and remove the rest",
			hm: hashMap{
				"PORT|Ethernet0": {
					"admin_status": "up",
				},
				"ASIC_STATE|SAI_OBJECT_TYPE_BRIDGE_PORT|oid|0x3a000000001a4a": {
					"SAI_BRIDGE_PORT_ATTR_ADMIN_STATE": "true",
				},
			},
			prefix: "PORT|",
			want: hashMap{
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
