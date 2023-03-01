package db

import (
	"context"
)

type AsicDB struct {
	c *Client
}

type OID string

func newAsicDB(addr string, id int, sep string) *AsicDB {
	return &AsicDB{
		c: NewClient(addr, id, sep),
	}
}

func (d *AsicDB) ExistRouterInterface(ctx context.Context, rif OID) (bool, error) {
	key := Key{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE", string(rif)}

	return d.c.Exists(ctx, key)
}

func (d *AsicDB) InFecModeRs(ctx context.Context, port OID) (bool, error) {
	key := Key{"ASIC_STATE", "SAI_OBJECT_TYPE_PORT", string(port)}

	result, err := d.c.HGet(ctx, key, "SAI_PORT_ATTR_FEC_MODE")
	if err != nil {
		return false, err
	}
	return result == "SAI_PORT_FEC_MODE_RS", err
}
