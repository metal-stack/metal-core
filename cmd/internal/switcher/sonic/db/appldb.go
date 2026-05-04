package db

import (
	"context"

	"github.com/valkey-io/valkey-go"
)

type ApplDB struct {
	c *Client
}

func newApplDB(rdb valkey.Client, sep string) *ApplDB {
	return &ApplDB{
		c: NewClient(rdb, sep),
	}
}

func (d *ApplDB) ExistPortInitDone(ctx context.Context) (bool, error) {
	key := Key{"PORT_TABLE", "PortInitDone"}

	return d.c.Exists(ctx, key)
}
