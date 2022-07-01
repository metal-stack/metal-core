package switcher

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

type ConfigDB struct {
	rdb       *redis.Client
	separator string
}

func NewConfigDB(cfg *SonicDatabaseConfig) *ConfigDB {
	db := cfg.Databases[configDB]
	i := cfg.Instances[db.Instance]
	rdb := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(i.Hostname, strconv.Itoa(i.Port)),
		DB:   db.Id,
	})
	return &ConfigDB{
		rdb:       rdb,
		separator: db.Separator,
	}
}

func (c *ConfigDB) SetEntry(key []string, values ...string) error {
	k := strings.Join(key, c.separator)
	if values == nil {
		return c.rdb.HSet(context.Background(), k, "NULL", "NULL").Err()
	}
	return c.rdb.HSet(context.Background(), k, values).Err()
}

func (c *ConfigDB) ModEntry(key []string, field string, value string) error {
	k := strings.Join(key, c.separator)
	val, err := c.rdb.HGet(context.Background(), k, field).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == redis.Nil || val != value {
		return c.rdb.HSet(context.Background(), k, field, value).Err()
	}
	return nil
}

func (c *ConfigDB) GetEntry(key []string) (map[string]string, error) {
	k := strings.Join(key, c.separator)
	return c.rdb.HGetAll(context.Background(), k).Result()
}

func (c *ConfigDB) DeleteEntry(key []string) error {
	k := strings.Join(key, c.separator)
	return c.rdb.Del(context.Background(), k).Err()
}

type View struct {
	keys      map[string]bool
	rdb       *redis.Client
	separator string
}

func (c *ConfigDB) GetView(table string) (*View, error) {
	p := table + c.separator
	keys, err := c.rdb.Keys(context.Background(), p+"*").Result()
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(keys))
	for _, key := range keys {
		set[key] = false
	}
	return &View{
		keys:      set,
		rdb:       c.rdb,
		separator: c.separator,
	}, nil
}

func (v *View) Contains(key []string) bool {
	k := strings.Join(key, v.separator)
	_, ok := v.keys[k]
	return ok
}

func (v *View) Mask(key []string) {
	k := strings.Join(key, v.separator)
	if _, ok := v.keys[k]; ok {
		v.keys[k] = true
	}
}

func (v *View) DeleteUnmasked() error {
	keys := make(map[string]bool)
	for key, masked := range v.keys {
		if masked {
			keys[key] = true
			continue
		}
		err := v.rdb.Del(context.Background(), key).Err()
		if err != nil {
			return err
		}
	}
	v.keys = keys
	return nil
}
