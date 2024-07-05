package db

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type CountersDB struct {
	c *Client
}

func newCountersDB(rdb *redis.Client, sep string) *CountersDB {
	return &CountersDB{
		c: NewClient(rdb, sep),
	}
}

func (d *CountersDB) GetPortNameMap(ctx context.Context) (map[string]OID, error) {
	val, err := d.c.HGetAll(ctx, Key{"COUNTERS_PORT_NAME_MAP"})
	return toOIDMap(val), err
}

func (d *CountersDB) GetRifNameMap(ctx context.Context) (map[string]OID, error) {
	val, err := d.c.HGetAll(ctx, Key{"COUNTERS_RIF_NAME_MAP"})
	return toOIDMap(val), err
}

func toOIDMap(val Val) map[string]OID {
	m := make(map[string]OID)
	for k, v := range val {
		m[k] = OID(v)
	}
	return m
}
