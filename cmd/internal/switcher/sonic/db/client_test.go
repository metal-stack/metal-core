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

func TestClient_Del(t *testing.T) {
	ctx := t.Context()
	sep := "|"
	vc := test.StartValkey(t)

	err := vc.Do(ctx, vc.B().Set().Key("table").Value("value1").Build()).Error()
	require.NoError(t, err)

	tests := []struct {
		name string
		key  Key
		want string
	}{
		{
			name: "delete non-existing",
			key:  Key{"some", "key"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			err := c.Del(ctx, tt.key)
			require.NoError(t, err)

			res := vc.Do(ctx, vc.B().Get().Key("table").Build())
			val, err := res.ToString()
			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, val); diff != "" {
				t.Errorf("Client.Del() diff = %s", diff)
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
