package db

import (
	"context"
)

type ApplDB struct {
	c *Client
}

func newApplDB(addr string, id int, sep string) *ApplDB {
	return &ApplDB{
		c: NewClient(addr, id, sep),
	}
}

func (d *ApplDB) ExistPortInitDone(ctx context.Context) (bool, error) {
	key := Key{"PORT_TABLE", "PortInitDone"}

	return d.c.Exists(ctx, key)
}
