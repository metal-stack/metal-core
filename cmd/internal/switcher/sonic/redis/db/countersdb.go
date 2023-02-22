package db

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type CountersDB struct {
	rdb       *redis.Client
	separator string
}

func newCountersDB(addr string, id int, separator string) *CountersDB {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       id,
		PoolSize: 1,
	})
	return &CountersDB{
		rdb:       rdb,
		separator: separator,
	}
}

func (c *CountersDB) GetOID(ctx context.Context, interfaceName string) (string, error) {
	rifNameMap, err := c.rdb.HGetAll(ctx, "COUNTERS_RIF_NAME_MAP").Result()
	if err != nil {
		return "", err
	}
	return rifNameMap[interfaceName], nil
}
