package db

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-redis/redismock/v9"
)

func TestAsicDB_GetPortIdBridgePortMap(t *testing.T) {
	db, mock := redismock.NewClientMock()
	asic := &AsicDB{c: &Client{rdb: db, sep: ":"}}
	bridgePort := "ASIC_STATE:SAI_OBJECT_TYPE_BRIDGE_PORT:oid:0x3a000000001a0a"
	want := map[OID]OID{OID("oid:0x1000000001251"): "oid:0x3a000000001a0a"}

	mock.ExpectKeys("ASIC_STATE:SAI_OBJECT_TYPE_BRIDGE_PORT:*").SetVal([]string{bridgePort})
	mock.ExpectHGet(bridgePort, "SAI_BRIDGE_PORT_ATTR_PORT_ID").SetVal("oid:0x1000000001251")

	got, err := asic.GetPortIdBridgePortMap(context.TODO())
	if err != nil {
		t.Errorf("GetPortIdBridgePortMap() error = %v, wantErr %v", err, nil)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetPortIdBridgePortMap() got = %v, want %v", got, want)
	}
}
