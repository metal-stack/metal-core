package switcher

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/go-redis/redis/v8"
)

const (
	configDB = "CONFIG_DB"
	stateDB  = "STATE_DB"
)

type SonicDatabaseConfig struct {
	Instances map[string]Instance `json:"INSTANCES"`
	Databases map[string]Database `json:"DATABASES"`
	Version   string              `json:"VERSION"`
}

type Instance struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
}

type Database struct {
	Id        int    `json:"id"`
	Separator string `json:"separator"`
	Instance  string `json:"instance"`
}

func NewClient(cfg *SonicDatabaseConfig) *redis.Client {
	i := cfg.Instances[cfg.Databases[configDB].Instance]
	rdb := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(i.Hostname, strconv.Itoa(i.Port)),
		DB:   cfg.Databases[configDB].Id,
	})

	return rdb
}

type ConfigDBApplier struct {
	rdb *redis.Client
	cfg *SonicDatabaseConfig
}

func NewConfigDBApplier(cfg *SonicDatabaseConfig) *ConfigDBApplier {
	return &ConfigDBApplier{
		rdb: NewClient(cfg),
		cfg: cfg,
	}
}

func (a *ConfigDBApplier) Apply(cfg *Conf) error {
	keys, err := a.rdb.Keys(context.Background(), "LOOPBACK_INTERFACE|*").Result()
	if err != nil {
		return err
	}

	infKey := "LOOPBACK_INTERFACE|Loopback0"
	ipKey := fmt.Sprintf("LOOPBACK_INTERFACE|Loopback0|%s/32", cfg.Loopback)
	infAlreadyConfigured := false
	ipAlreadyConfigured := false
	toBeDeleted := make([]string, 0)
	for _, key := range keys {
		switch key {
		case infKey:
			infAlreadyConfigured = true
		case ipKey:
			ipAlreadyConfigured = true
		default:
			toBeDeleted = append(toBeDeleted, key)
		}
	}

	if len(toBeDeleted) > 0 {
		for _, key := range toBeDeleted {
			err = a.rdb.Del(context.Background(), key).Err()
			if err != nil {
				return err
			}
		}
	}

	if !infAlreadyConfigured {
		err = a.rdb.HSet(context.Background(), infKey, "NULL", "NULL").Err()
		if err != nil {
			return err
		}
	}
	if !ipAlreadyConfigured {
		err = a.rdb.HSet(context.Background(), ipKey, "NULL", "NULL").Err()
		if err != nil {
			return err
		}
	}
	return nil
}
