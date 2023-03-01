package db

import (
	"context"
)

type AsicDB struct {
	c *Client
}

func newAsicDB(addr string, id int, sep string) *AsicDB {
	return &AsicDB{
		c: NewClient(addr, id, sep),
	}
}

func (d *AsicDB) ExistRouterInterface(ctx context.Context, oid string) (bool, error) {
	key := Key{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE", oid}

	return d.c.Exists(ctx, key)
}
