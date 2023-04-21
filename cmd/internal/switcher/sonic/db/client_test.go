package db

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-redis/redismock/v9"
)

func NewClientMock(sep string) (*Client, redismock.ClientMock) {
	db, mock := redismock.NewClientMock()
	c := &Client{
		rdb: db,
		sep: sep,
	}
	return c, mock
}

func TestClient_Del(t *testing.T) {
	c, mock := NewClientMock("|")

	mock.ExpectDel("table|entry").SetVal(1)

	if err := c.Del(context.Background(), Key{"table", "entry"}); err != nil {
		t.Errorf("Del() error = %v, wantErr %v", err, false)
	}
}

func TestClient_Exists(t *testing.T) {
	tests := []struct {
		name string
		val  int64
		want bool
	}{
		{
			name: "Doesn't exist",
			val:  0,
			want: false,
		}, {
			name: "Exists",
			val:  1,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mock := NewClientMock("|")

			mock.ExpectExists("table|key").SetVal(tt.val)

			got, err := c.Exists(context.Background(), Key{"table", "key"})
			if err != nil {
				t.Errorf("Exists() error = %v, wantErr %v", err, false)
				return
			}
			if got != tt.want {
				t.Errorf("Exists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetTable(t *testing.T) {
	c, _ := NewClientMock("|")
	want := &Table{
		client: c,
		name:   "table|sub",
	}

	if got := c.GetTable(Key{"table", "sub"}); !reflect.DeepEqual(got, want) {
		t.Errorf("GetTable() = %v, want %v", got, want)
	}
}

func TestClient_GetView(t *testing.T) {
	c, mock := NewClientMock("|")
	want := View{"key": {}, "key1|key2": {}}

	mock.ExpectKeys("table|*").SetVal([]string{"table|key", "table|key1|key2"})

	got, err := c.GetView(context.Background(), "table")
	if err != nil {
		t.Errorf("GetView() error = %v, wantErr %v", err, false)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetView() got = %v, want %v", got, want)
	}
}

func TestClient_HGet(t *testing.T) {
	c, mock := NewClientMock("|")

	mock.ExpectHGet("table|key", "field").RedisNil()

	got, err := c.HGet(context.Background(), Key{"table", "key"}, "field")
	if err != nil {
		t.Errorf("HGet() error = %v, wantErr %v", err, false)
		return
	}
	if len(got) != 0 {
		t.Errorf("HGet() got = %v, want %v", got, "")
	}
}

func TestClient_HGetAll(t *testing.T) {
	c, mock := NewClientMock("|")
	want := Val{"key": "test"}

	mock.ExpectHGetAll("table|key").SetVal(map[string]string{"key": "test"})

	got, err := c.HGetAll(context.Background(), Key{"table", "key"})
	if err != nil {
		t.Errorf("HGetAll() error = %v, wantErr %v", err, false)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("HGetAll() got = %v, want %v", got, want)
	}
}

func TestClient_HSet(t *testing.T) {
	c, mock := NewClientMock("|")

	val := Val{"key": "test"}
	mock.ExpectHSet("table|key", "key", "test").SetVal(1)

	err := c.HSet(context.Background(), Key{"table", "key"}, val)
	if err != nil {
		t.Errorf("HSet() error = %v, wantErr %v", err, false)
	}
}

func TestClient_Keys(t *testing.T) {
	c, mock := NewClientMock("|")
	want := []Key{
		{"table", "key"},
		{"table", "key1", "key2"},
	}

	mock.ExpectKeys("table|*").SetVal([]string{"table|key", "table|key1|key2"})

	got, err := c.Keys(context.Background(), Key{"table", "*"})
	if err != nil {
		t.Errorf("Keys() error = %v, wantErr %v", err, false)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Keys() got = %v, want %v", got, want)
	}
}
