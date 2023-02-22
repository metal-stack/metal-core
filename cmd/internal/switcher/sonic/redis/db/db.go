package db

type Config struct {
	Databases map[string]database `json:"DATABASES"`
	Instances map[string]instance `json:"INSTANCES"`
}

type database struct {
	Id        int    `json:"id"`
	Instance  string `json:"instance"`
	Separator string `json:"separator"`
}

type instance struct {
	Addr string `json:"unix_socket_path"`
}

type DB struct {
	Config *ConfigDB
	State  *StateDB
}

func New(cfg *Config) *DB {
	configDB := cfg.Databases["CONFIG_DB"]
	stateDB := cfg.Databases["STATE_DB"]

	return &DB{
		Config: newConfigDB(cfg.Instances[configDB.Instance].Addr, configDB.Id, configDB.Separator),
		State:  newStateDB(cfg.Instances[stateDB.Instance].Addr, stateDB.Id, stateDB.Separator),
	}
}
