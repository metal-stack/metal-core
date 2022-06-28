package switcher

import (
	"context"
	"net"
	"strconv"
	"strings"

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

	lo := cfg.Loopback + "/32"
	infAlreadyConfigured := false
	ipAlreadyConfigured := false
	toBeDeleted := make([]string, 0)
	for _, key := range keys {
		s := strings.Split(key, "|")
		if len(s) == 2 && s[1] == "Loopback0" {
			infAlreadyConfigured = true
		} else if len(s) == 2 && s[1] != "Loopback0" {
			toBeDeleted = append(toBeDeleted, key)
		} else if len(s) == 3 && s[2] != lo {
			toBeDeleted = append(toBeDeleted, key)
		} else if len(s) == 3 && s[2] == lo {
			ipAlreadyConfigured = true
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
		err = a.rdb.HSet(context.Background(), "LOOPBACK_INTERFACE|Loopback0", "NULL", "NULL").Err()
		if err != nil {
			return err
		}
	}
	if !ipAlreadyConfigured {
		err = a.rdb.HSet(context.Background(), "LOOPBACK_INTERFACE|Loopback0|"+lo, "NULL", "NULL").Err()
		if err != nil {
			return err
		}
	}
	return nil
}
