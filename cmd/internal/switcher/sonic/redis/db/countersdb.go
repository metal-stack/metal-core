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
	rifNameMap, err := c.c.HGetAll(ctx, Key{"COUNTERS_RIF_NAME_MAP"})
	if err != nil {
		return "", err
	}
	return rifNameMap[interfaceName], nil
}
