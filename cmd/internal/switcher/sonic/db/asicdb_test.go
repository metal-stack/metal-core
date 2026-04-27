package db

import (
	"context"
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
			name: "",
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
	type fields struct {
		c *Client
	}
	type args struct {
		ctx        context.Context
		bridgePort OID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &AsicDB{
				c: tt.fields.c,
			}
			got, err := d.ExistBridgePort(tt.args.ctx, tt.args.bridgePort)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsicDB.ExistBridgePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AsicDB.ExistBridgePort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsicDB_ExistRouterInterface(t *testing.T) {
	type fields struct {
		c *Client
	}
	type args struct {
		ctx context.Context
		rif OID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &AsicDB{
				c: tt.fields.c,
			}
			got, err := d.ExistRouterInterface(tt.args.ctx, tt.args.rif)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsicDB.ExistRouterInterface() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AsicDB.ExistRouterInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}
