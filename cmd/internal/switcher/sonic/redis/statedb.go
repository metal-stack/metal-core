package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type stateDB struct {
	rdb       *redis.Client
	separator string
}

func newStateDB(addr string, id int, separator string) *stateDB {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       id,
		PoolSize: 1,
	})
	return &stateDB{
		rdb:       rdb,
		separator: separator,
	}
}

func (s *stateDB) existInterface(ctx context.Context, interfaceName string) (bool, error) {
	key := "INTERFACE_TABLE" + s.separator + interfaceName

	result, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result != 0, nil
}
