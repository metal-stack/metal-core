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
	Appl     *ApplDB
	Asic     *AsicDB
	Config   *ConfigDB
	Counters *CountersDB
}

func New(cfg *Config) *DB {
	applDB := cfg.Databases["APPL_DB"]
	asicDB := cfg.Databases["ASIC_DB"]
	configDB := cfg.Databases["CONFIG_DB"]
	countersDB := cfg.Databases["COUNTERS_DB"]

	return &DB{
		Appl:     newApplDB(cfg.Instances[applDB.Instance].Addr, applDB.Id, applDB.Separator),
		Asic:     newAsicDB(cfg.Instances[asicDB.Instance].Addr, asicDB.Id, asicDB.Separator),
		Config:   newConfigDB(cfg.Instances[configDB.Instance].Addr, configDB.Id, configDB.Separator),
		Counters: newCountersDB(cfg.Instances[countersDB.Instance].Addr, countersDB.Id, countersDB.Separator),
	}
}
