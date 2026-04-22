package db

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

var (
	testData = test.StringMap{
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
		mods      func(test.StringMap)
		key       Key
		separator string
	}{
		{
			name:      "delete non-existing",
			data:      testData,
			mods:      func(test.StringMap) {},
			key:       Key{"some", "key"},
			separator: "|",
		},
		{
			name: "delete existing",
			data: testData,
			mods: func(data test.StringMap) {
				delete(data["PORT"].(test.StringMap), "Ethernet0")
			},
			key:       Key{"PORT", "Ethernet0"},
			separator: "|",
		},
		{
			name: "delete last entry for key",
			data: testData,
			mods: func(data test.StringMap) {
				delete(data, "ASIC_STATE")
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

			c := &Client{
				rdb: vc,
				sep: tt.separator,
			}
			err = c.Del(ctx, tt.key)
			require.NoError(t, err)

			data, err := test.GetData(ctx, vc, tt.separator)
			require.NoError(t, err)

			if tt.mods != nil {
				tt.mods(tt.data)
			}
			if diff := cmp.Diff(tt.data, data); diff != "" {
				t.Errorf("Client.Del() data differs = %s", diff)
			}
		})
	}
}

func TestClient_Exists(t *testing.T) {
	type fields struct {
		rdb valkey.Client
		sep string
	}
	type args struct {
		ctx context.Context
		key Key
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
			c := &Client{
				rdb: tt.fields.rdb,
				sep: tt.fields.sep,
			}
			got, err := c.Exists(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetTable(t *testing.T) {
	type fields struct {
		rdb valkey.Client
		sep string
	}
	type args struct {
		table Key
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Table
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				rdb: tt.fields.rdb,
				sep: tt.fields.sep,
			}
			if got := c.GetTable(tt.args.table); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.GetTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetView(t *testing.T) {
	type fields struct {
		rdb valkey.Client
		sep string
	}
	type args struct {
		ctx   context.Context
		table string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    View
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				rdb: tt.fields.rdb,
				sep: tt.fields.sep,
			}
			got, err := c.GetView(tt.args.ctx, tt.args.table)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetView() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.GetView() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_HGet(t *testing.T) {
	type fields struct {
		rdb valkey.Client
		sep string
	}
	type args struct {
		ctx   context.Context
		key   Key
		field string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				rdb: tt.fields.rdb,
				sep: tt.fields.sep,
			}
			got, err := c.HGet(tt.args.ctx, tt.args.key, tt.args.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.HGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.HGet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_HGetAll(t *testing.T) {
	type fields struct {
		rdb valkey.Client
		sep string
	}
	type args struct {
		ctx context.Context
		key Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Val
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				rdb: tt.fields.rdb,
				sep: tt.fields.sep,
			}
			got, err := c.HGetAll(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.HGetAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.HGetAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_HSet(t *testing.T) {
	type fields struct {
		rdb valkey.Client
		sep string
	}
	type args struct {
		ctx context.Context
		key Key
		val Val
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				rdb: tt.fields.rdb,
				sep: tt.fields.sep,
			}
			if err := c.HSet(tt.args.ctx, tt.args.key, tt.args.val); (err != nil) != tt.wantErr {
				t.Errorf("Client.HSet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Keys(t *testing.T) {
	type fields struct {
		rdb valkey.Client
		sep string
	}
	type args struct {
		ctx     context.Context
		pattern Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Key
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				rdb: tt.fields.rdb,
				sep: tt.fields.sep,
			}
			got, err := c.Keys(tt.args.ctx, tt.args.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Keys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.Keys() = %v, want %v", got, tt.want)
			}
		})
	}
}
