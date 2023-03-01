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

func (c *CountersDB) GetOID(ctx context.Context, interfaceName string) (string, error) {
	return c.c.HGet(ctx, Key{"COUNTERS_RIF_NAME_MAP"}, interfaceName)
}
