package db

import "github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis"

type DB struct {
	Config *ConfigDB
	State  *StateDB
}

func New(cfg *redis.Config) *DB {
	configDB := cfg.Databases["CONFIG_DB"]
	stateDB := cfg.Databases["STATE_DB"]

	return &DB{
		Config: newConfigDB(cfg.Instances[configDB.Instance].Addr, configDB.Id, configDB.Separator),
		State:  newStateDB(cfg.Instances[stateDB.Instance].Addr, stateDB.Id, stateDB.Separator),
	}
}
