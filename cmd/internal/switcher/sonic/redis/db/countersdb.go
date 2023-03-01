package db

import (
	"context"
)

type CountersDB struct {
	c *Client
}

func newCountersDB(addr string, id int, sep string) *CountersDB {
	return &CountersDB{
		c: NewClient(addr, id, sep),
	}
}

func (d *CountersDB) GetOID(ctx context.Context, interfaceName string) (OID, error) {
	oid, err := d.c.HGet(ctx, Key{"COUNTERS_RIF_NAME_MAP"}, interfaceName)
	return OID(oid), err
}

func (d *CountersDB) GetPortNameMap(ctx context.Context) (map[string]OID, error) {
	result, err := d.c.HGetAll(ctx, Key{"COUNTERS_PORT_NAME_MAP"})
	m := make(map[string]OID)
	for k, v := range result {
		m[k] = OID(v)
	}
	return m, err
}
