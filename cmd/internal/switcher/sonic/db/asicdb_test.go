package db

import (
	"context"
	"reflect"
	"testing"
)

func TestAsicDB_GetPortIdBridgePortMap(t *testing.T) {
	type fields struct {
		c *Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[OID]OID
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &AsicDB{
				c: tt.fields.c,
			}
			got, err := d.GetPortIdBridgePortMap(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsicDB.GetPortIdBridgePortMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AsicDB.GetPortIdBridgePortMap() = %v, want %v", got, tt.want)
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
