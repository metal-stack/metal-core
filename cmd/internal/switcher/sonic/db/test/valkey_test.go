package test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

type (
	hashMap map[string]map[string]string
)

func TestLoadData(t *testing.T) {

	tests := []struct {
		name string
		d    data
		want hashMap
	}{
		{
			name: "empty data",
			d:    data{},
			want: nil,
		},
		{
			name: "add empty fields and values to key",
			d: data{
				"LOOPBACK_INTERFACE": {
					"Loopback0": {},
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
			d: data{
				"PORT": {
					"Ethernet0": {
						"admin_status": "up",
						"mtu":          "9000",
					},
					"Ethernet1": {
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

			err := LoadData(ctx, vc, tt.d, sep)
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
	var (
		ctx = t.Context()
	)

	tests := []struct {
		name       string
		separator  string
		beforeFunc func(valkey.Client)
		want       data
	}{
		{
			name:       "empty data",
			separator:  "|",
			beforeFunc: func(valkey.Client) {},
			want:       data{},
		},
		{
			name:      "load NULL correctly",
			separator: "|",
			beforeFunc: func(vc valkey.Client) {
				d := data{
					"LOOPBACK_INTERFACE": {
						"Loopback0": {},
					},
				}
				err := LoadData(ctx, vc, d, "|")
				require.NoError(t, err)
			},
			want: data{
				"LOOPBACK_INTERFACE": {
					"Loopback0": {},
				},
			},
		},
		{
			name:      "load all values correctly",
			separator: "|",
			beforeFunc: func(vc valkey.Client) {
				d := data{
					"PORT": {
						"Ethernet0": {
							"admin_status": "up",
							"mtu":          "9000",
						},
						"Ethernet1": {
							"speed": "25000",
							"alias": "Eth1/2",
						},
					},
				}
				err := LoadData(ctx, vc, d, "|")
				require.NoError(t, err)
			},
			want: data{
				"PORT": {
					"Ethernet0": {
						"admin_status": "up",
						"mtu":          "9000",
					},
					"Ethernet1": {
						"speed": "25000",
						"alias": "Eth1/2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				vc = StartValkey(t)
			)
			defer vc.Close()

			if tt.beforeFunc != nil {
				tt.beforeFunc(vc)
			}

			got, err := GetData(ctx, vc, tt.separator)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetData() diff = %s", diff)
			}
		})
	}
}
