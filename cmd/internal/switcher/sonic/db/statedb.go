package db

import (
	"context"

	"github.com/valkey-io/valkey-go"
)

type StateDB struct {
	c *Client
}

func newStateDB(rdb valkey.Client, sep string) *StateDB {
	return &StateDB{
		c: NewClient(rdb, sep),
	}
}

// Dump reads all entries of the tables from the STATE_DB.
func (d *StateDB) Dump(ctx context.Context, tables ...Key) (Dump, error) {
	return d.c.Dump(ctx, tables...)
}
