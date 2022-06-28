package switcher

import (
	"reflect"
	"testing"

	"github.com/go-redis/redismock/v8"
)

func TestConfigDB_DeleteEntry(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	db := &ConfigDB{
		rdb:       rdb,
		separator: ":",
	}

	mock.ExpectDel("table:key").SetVal(1)

	if err := db.DeleteEntry([]string{"table", "key"}); err != nil {
		t.Errorf("DeleteEntry() unexpected error = %v", err)
		return
	}
}

func TestConfigDB_GetEntry(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	db := &ConfigDB{
		rdb:       rdb,
		separator: ":",
	}

	expected := map[string]string{"key": "value"}
	mock.ExpectHGetAll("table:key").SetVal(expected)

	got, err := db.GetEntry([]string{"table", "key"})
	if err != nil {
		t.Errorf("GetEntry() unexpected error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("GetEntry() got = %v, want %v", got, expected)
	}
}

func TestConfigDB_SetEntry(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	db := &ConfigDB{
		rdb:       rdb,
		separator: ":",
	}

	mock.ExpectHSet("table:key", "field", "value").SetVal(1)

	err := db.SetEntry([]string{"table", "key"}, "field", "value")
	if err != nil {
		t.Errorf("SetEntry() unexpected error = %v", err)
		return
	}
}

func TestConfigDB_SetEntry_NilValues(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	db := &ConfigDB{
		rdb:       rdb,
		separator: ":",
	}

	mock.ExpectHSet("table:key", "NULL", "NULL").SetVal(1)

	err := db.SetEntry([]string{"table", "key"})
	if err != nil {
		t.Errorf("SetEntry() unexpected error = %v", err)
		return
	}
}

func TestConfigDB_GetView(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	db := &ConfigDB{
		rdb:       rdb,
		separator: "|",
	}

	expected := &View{
		keys:      map[string]bool{"key1": false, "key2": false},
		rdb:       rdb,
		separator: "|",
		table:     "table",
	}
	mock.ExpectKeys("table|*").SetVal([]string{"key1", "key2"})

	got, err := db.GetView("table")
	if err != nil {
		t.Errorf("GetView() unexpected error %v", err)
		return
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("GetView() got = %v, want %v", got, expected)
	}
}

func TestView_DeleteUnmasked(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	view := &View{
		keys: map[string]bool{"table|key1": true, "table|key2": false},
		rdb:  rdb,
	}

	expected := map[string]bool{"table|key1": true}
	mock.ExpectDel("table|key2").SetVal(1)

	err := view.DeleteUnmasked()
	if err != nil {
		t.Errorf("DeleteUnmasked() unexpected error %v", err)
	}
	if !reflect.DeepEqual(view.keys, expected) {
		t.Errorf("DeleteUnmasked() got = %v, want %v", view.keys, expected)
	}
}
