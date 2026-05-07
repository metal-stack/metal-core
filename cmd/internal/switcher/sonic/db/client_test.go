package db

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

var (
	clientTestData = test.StringMap{
		"LOOPBACK_INTERFACE": test.StringMap{
			"Loopback0": test.StringMap{},
		},
		"PORT": test.StringMap{
			"Ethernet0": test.StringMap{
				"admin_status": "up",
				"alias":        "Eth1/1",
			},
			"Ethernet1": test.StringMap{
				"admin_status": "up",
				"alias":        "Eth1/2",
			},
		},
		"ASIC_STATE": test.StringMap{
			"SAI_OBJECT_TYPE_BRIDGE_PORT": test.StringMap{
				"oid": test.StringMap{
					"0x3a000000001a4a": test.StringMap{
						"SAI_BRIDGE_PORT_ATTR_ADMIN_STATE": "true",
					},
				},
			},
		},
	}
)

func TestClient_Del(t *testing.T) {
	tests := []struct {
		name      string
		data      test.StringMap
		mods      func(test.HashMap)
		key       Key
		separator string
	}{
		{
			name:      "delete non-existing",
			data:      clientTestData,
			mods:      func(test.HashMap) {},
			key:       Key{"some", "key"},
			separator: "|",
		},
		{
			name: "delete existing",
			data: clientTestData,
			mods: func(data test.HashMap) {
				delete(data, "PORT|Ethernet0")
			},
			key:       Key{"PORT", "Ethernet0"},
			separator: "|",
		},
		{
			name: "delete last entry for key",
			data: clientTestData,
			mods: func(data test.HashMap) {
				delete(data, "ASIC_STATE:SAI_OBJECT_TYPE_BRIDGE_PORT:oid:0x3a000000001a4a")
			},
			key:       Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", "oid", "0x3a000000001a4a"},
			separator: ":",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, tt.separator)
			require.NoError(t, err)
			initData, err := test.GetData(ctx, vc, tt.separator)
			require.NoError(t, err)
			if tt.mods != nil {
				tt.mods(initData)
			}

			c := &Client{
				rdb: vc,
				sep: tt.separator,
			}
			err = c.Del(ctx, tt.key)
			require.NoError(t, err)

			data, err := test.GetData(ctx, vc, tt.separator)
			require.NoError(t, err)
			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("Client.Del() data differs = %s", diff)
			}
		})
	}
}

func TestClient_Exists(t *testing.T) {
	tests := []struct {
		name      string
		data      test.StringMap
		key       Key
		separator string
		want      bool
	}{
		{
			name:      "not existing",
			data:      clientTestData,
			key:       Key{"some", "key"},
			separator: "|",
			want:      false,
		},
		{
			name:      "existing",
			data:      clientTestData,
			key:       Key{"LOOPBACK_INTERFACE", "Loopback0"},
			separator: "|",
			want:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, tt.separator)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: tt.separator,
			}

			got, err := c.Exists(ctx, tt.key)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("Client.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetView(t *testing.T) {
	tests := []struct {
		name      string
		table     string
		data      test.StringMap
		separator string
		want      View
	}{
		{
			name:      "table does not exist",
			table:     "some key",
			data:      clientTestData,
			separator: "|",
			want:      View{},
		},
		{
			name:      "table exists but empty",
			table:     "LOOPBACK_INTERFACE|Loopback0",
			data:      clientTestData,
			separator: "|",
			want:      View{},
		},
		{
			name:      "table exists",
			table:     "PORT",
			data:      clientTestData,
			separator: "|",
			want: View{
				"Ethernet0": struct{}{},
				"Ethernet1": struct{}{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, tt.separator)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: tt.separator,
			}

			got, err := c.GetView(ctx, tt.table)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Client.GetView() diff = %s", diff)
			}
		})
	}
}

func TestClient_HGet(t *testing.T) {
	tests := []struct {
		name      string
		data      test.StringMap
		separator string
		key       Key
		field     string
		want      string
	}{
		{
			name:      "not existing",
			data:      clientTestData,
			separator: "|",
			key:       Key{"some", "key"},
			field:     "some_field",
			want:      "",
		},
		{
			name:      "get empty",
			data:      clientTestData,
			separator: "|",
			key:       Key{"LOOPBACK_INTERFACE", "Loopback0"},
			field:     "NULL",
			want:      "NULL",
		},
		{
			name:      "get partial key",
			data:      clientTestData,
			separator: ":",
			key:       Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", "oid"},
			field:     "0x3a000000001a4a",
			want:      "",
		},
		{
			name:      "get existing",
			data:      clientTestData,
			separator: ":",
			key:       Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", "oid", "0x3a000000001a4a"},
			field:     "SAI_BRIDGE_PORT_ATTR_ADMIN_STATE",
			want:      "true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, tt.separator)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: tt.separator,
			}
			got, err := c.HGet(ctx, tt.key, tt.field)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("Client.HGet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_HGetAll(t *testing.T) {
	tests := []struct {
		name      string
		data      test.StringMap
		separator string
		key       Key
		want      Val
	}{
		{
			name:      "not existing",
			data:      clientTestData,
			separator: "|",
			key:       Key{"some", "key"},
			want:      Val{},
		},
		{
			name:      "get empty",
			data:      clientTestData,
			separator: "|",
			key:       Key{"LOOPBACK_INTERFACE", "Loopback0"},
			want: Val{
				"NULL": "NULL",
			},
		},
		{
			name:      "get partial key",
			data:      clientTestData,
			separator: "|",
			key:       Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", "oid"},
			want:      Val{},
		},
		{
			name:      "get existing",
			data:      clientTestData,
			separator: ":",
			key:       Key{"PORT", "Ethernet1"},
			want: Val{
				"admin_status": "up",
				"alias":        "Eth1/2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run(tt.name, func(t *testing.T) {
				var (
					ctx = t.Context()
					vc  = test.StartValkey(t)
				)
				defer vc.Close()

				err := test.LoadData(ctx, vc, tt.data, tt.separator)
				require.NoError(t, err)

				c := &Client{
					rdb: vc,
					sep: tt.separator,
				}
				got, err := c.HGetAll(ctx, tt.key)
				require.NoError(t, err)
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("Client.HGet() diff = %s", diff)
				}
			})
		})
	}
}

func TestClient_HSet(t *testing.T) {
	tests := []struct {
		name      string
		data      test.StringMap
		mods      func(test.HashMap)
		separator string
		key       Key
		val       Val
	}{
		{
			name: "set value for new key",
			data: clientTestData,
			mods: func(data test.HashMap) {
				data["some|new|key"] = map[string]string{
					"some_field": "some_value",
				}
			},
			separator: "|",
			key:       Key{"some", "new", "key"},
			val: Val{
				"some_field": "some_value",
			},
		},
		{
			name: "set value for existing key",
			data: clientTestData,
			mods: func(data test.HashMap) {
				data["PORT|Ethernet0"]["admin_status"] = "down"
			},
			separator: "|",
			key:       Key{"PORT", "Ethernet0"},
			val: Val{
				"admin_status": "down",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, tt.separator)
			require.NoError(t, err)
			initData, err := test.GetData(ctx, vc, tt.separator)
			require.NoError(t, err)
			if tt.mods != nil {
				tt.mods(initData)
			}

			c := &Client{
				rdb: vc,
				sep: tt.separator,
			}

			err = c.HSet(ctx, tt.key, tt.val)
			require.NoError(t, err)

			data, err := test.GetData(ctx, vc, tt.separator)
			require.NoError(t, err)
			if diff := cmp.Diff(initData, data); diff != "" {
				t.Errorf("Client.HSet() data differs = %s", diff)
			}
		})
	}
}

func TestClient_Keys(t *testing.T) {
	tests := []struct {
		name      string
		data      test.StringMap
		separator string
		pattern   Key
		want      []Key
	}{
		{
			name:      "get all keys",
			data:      clientTestData,
			separator: "|",
			pattern:   Key{"*"},
			want: []Key{
				{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", "oid", "0x3a000000001a4a"},
				{"LOOPBACK_INTERFACE", "Loopback0"},
				{"PORT", "Ethernet0"},
				{"PORT", "Ethernet1"},
			},
		},
		{
			name:      "get all keys with prefix",
			data:      clientTestData,
			separator: "|",
			pattern:   Key{"PORT|*"},
			want: []Key{
				{"PORT", "Ethernet0"},
				{"PORT", "Ethernet1"},
			},
		},
		{
			name:      "get non-existing",
			data:      clientTestData,
			separator: "|",
			pattern:   Key{"VLAN|*"},
			want:      []Key{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, tt.separator)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: tt.separator,
			}
			got, err := c.Keys(ctx, tt.pattern)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Client.Keys() diff = %s", diff)
			}
		})
	}
}
