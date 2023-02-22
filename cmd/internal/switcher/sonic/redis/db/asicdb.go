package db

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type AsicDB struct {
	rdb       *redis.Client
	separator string
}

func newAsicDB(addr string, id int, separator string) *AsicDB {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       id,
		PoolSize: 1,
	})
	return &AsicDB{
		rdb:       rdb,
		separator: separator,
	}
}

func (a *AsicDB) ExistRouterInterface(ctx context.Context, oid string) (bool, error) {
	key := "ASIC_STATE" + a.separator + "SAI_OBJECT_TYPE_ROUTER_INTERFACE" + a.separator + oid

	result, err := a.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result != 0, nil
}
