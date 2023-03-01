package db

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-redis/redismock/v9"
)

func TestClient_Del(t *testing.T) {
	db, mock := redismock.NewClientMock()
	c := &Client{rdb: db, sep: "|"}

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
			db, mock := redismock.NewClientMock()
			c := &Client{rdb: db, sep: "|"}

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

func TestClient_HGetAll(t *testing.T) {
	db, mock := redismock.NewClientMock()
	c := &Client{rdb: db, sep: "|"}
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
	db, mock := redismock.NewClientMock()
	c := &Client{rdb: db, sep: "|"}

	val := Val{"key": "test"}
	mock.ExpectHSet("table|key", "key", "test").SetVal(1)

	err := c.HSet(context.Background(), Key{"table", "key"}, val)
	if err != nil {
		t.Errorf("HSet() error = %v, wantErr %v", err, false)
	}
}

func TestClient_Keys(t *testing.T) {
	db, mock := redismock.NewClientMock()
	c := &Client{rdb: db, sep: "|"}
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
