package db

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

func TestAsicDB_GetPortIdBridgePortMap(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		want map[OID]OID
	}{
		{
			name: "get non-empty port ids",
			data: test.StringMap{
				"ASIC_STATE": test.StringMap{
					"SAI_OBJECT_TYPE_BRIDGE_PORT": test.StringMap{
						"oid": test.StringMap{
							"0x3a000000000daa": test.StringMap{
								"SAI_BRIDGE_PORT_ATTR_PORT_ID": "oid:0x3a000000000daa",
							},
							"0x3a000000000dab": test.StringMap{
								"SAI_BRIDGE_PORT_ATTR_PORT_ID": "oid:0x3a000000000dab",
							},
							"0x3a000000000dac": test.StringMap{},
						},
					},
				},
			},
			want: map[OID]OID{
				"oid:0x3a000000000daa": "oid:0x3a000000000daa",
				"oid:0x3a000000000dab": "oid:0x3a000000000dab",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = ":"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &AsicDB{
				c: c,
			}
			got, err := d.GetPortIdBridgePortMap(ctx)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("AsicDB.GetPortIdBridgePortMap() diff = %s", diff)
			}
		})
	}
}

func TestAsicDB_ExistBridgePort(t *testing.T) {
	tests := []struct {
		name       string
		data       test.StringMap
		bridgePort OID
		want       bool
	}{
		{
			name: "not exists",
			data: test.StringMap{
				"ASIC_STATE": test.StringMap{
					"SAI_OBJECT_TYPE_BRIDGE_PORT": test.StringMap{
						"oid": test.StringMap{
							"0x3a000000000dac": test.StringMap{},
						},
					},
				},
			},
			bridgePort: OID("oid:0x3a000000000dac"),
			want:       false,
		},
		{
			name: "exists",
			data: test.StringMap{
				"ASIC_STATE": test.StringMap{
					"SAI_OBJECT_TYPE_BRIDGE_PORT": test.StringMap{
						"oid": test.StringMap{
							"0x3a000000000daa": test.StringMap{
								"SAI_BRIDGE_PORT_ATTR_PORT_ID": "oid:0x3a000000000daa",
							},
						},
					},
				},
			},
			bridgePort: OID("oid:0x3a000000000daa"),
			want:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = ":"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &AsicDB{
				c: c,
			}
			got, err := d.ExistBridgePort(ctx, tt.bridgePort)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("AsicDB.ExistBridgePort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsicDB_ExistRouterInterface(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		rif  OID
		want bool
	}{
		{
			name: "not exists",
			data: test.StringMap{
				"ASIC_STATE": test.StringMap{
					"SAI_OBJECT_TYPE_ROUTER_INTERFACE": test.StringMap{
						"oid": test.StringMap{
							"0x3a000000000dac": test.StringMap{},
						},
					},
				},
			},
			rif:  "oid:0x3a000000000dac",
			want: false,
		},
		{
			name: "exists",
			data: test.StringMap{
				"ASIC_STATE": test.StringMap{
					"SAI_OBJECT_TYPE_ROUTER_INTERFACE": test.StringMap{
						"oid": test.StringMap{
							"0x3a000000000dac": test.StringMap{
								"SAI_ROUTER_INTERFACE_ATTR_TYPE": "SAI_ROUTER_INTERFACE_TYPE_PORT",
							},
						},
					},
				},
			},
			rif:  "oid:0x3a000000000dac",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = ":"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &AsicDB{
				c: c,
			}
			got, err := d.ExistRouterInterface(ctx, tt.rif)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("AsicDB.ExistRouterInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}
