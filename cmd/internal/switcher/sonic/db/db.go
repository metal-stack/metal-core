package db

import (
	"fmt"
	"os"

	"github.com/valkey-io/valkey-go"
)

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
	Addr         string `json:"unix_socket_path"`
	PasswordPath string `json:"password_path"`
}

type DB struct {
	Appl     *ApplDB
	Asic     *AsicDB
	Config   *ConfigDB
	Counters *CountersDB
}

func New(cfg *Config) (*DB, error) {
	applDB := cfg.Databases["APPL_DB"]
	asicDB := cfg.Databases["ASIC_DB"]
	configDB := cfg.Databases["CONFIG_DB"]
	countersDB := cfg.Databases["COUNTERS_DB"]

	applClient, err := newRedisClient(cfg.Instances[applDB.Instance], applDB.Id)
	if err != nil {
		return nil, fmt.Errorf("could not create client for APPL_DB: %w", err)
	}

	asicClient, err := newRedisClient(cfg.Instances[asicDB.Instance], asicDB.Id)
	if err != nil {
		return nil, fmt.Errorf("could not create client for ASIC_DB: %w", err)
	}

	configClient, err := newRedisClient(cfg.Instances[configDB.Instance], configDB.Id)
	if err != nil {
		return nil, fmt.Errorf("could not create client for CONFIG_DB: %w", err)
	}

	countersClient, err := newRedisClient(cfg.Instances[countersDB.Instance], countersDB.Id)
	if err != nil {
		return nil, fmt.Errorf("could not create client for COUNTERS_DB: %w", err)
	}

	db := &DB{
		Appl:     newApplDB(applClient, applDB.Separator),
		Asic:     newAsicDB(asicClient, asicDB.Separator),
		Config:   newConfigDB(configClient, configDB.Separator),
		Counters: newCountersDB(countersClient, countersDB.Separator),
	}
	return db, nil
}

func newRedisClient(redisInstance instance, redisDatabase int) (valkey.Client, error) {
	if redisInstance.PasswordPath != "" {
		return newRedisClientWithAuth(redisInstance, redisDatabase)
	}

	valkeyClient, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{redisInstance.Addr},
		SelectDB:    redisDatabase,
		ClientName:  "metal-apiserver",
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create valkey client: %w", err)
	}

	return valkeyClient, nil
}

func newRedisClientWithAuth(redisInstance instance, redisDatabase int) (valkey.Client, error) {
	passwd, err := os.ReadFile(redisInstance.PasswordPath)
	if err != nil {
		return nil, fmt.Errorf("could not read password from %s: %w", redisInstance.PasswordPath, err)
	}
	valkeyClient, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{redisInstance.Addr},
		AuthCredentialsFn: func(acc valkey.AuthCredentialsContext) (valkey.AuthCredentials, error) {
			return valkey.AuthCredentials{
				Password: string(passwd),
			}, nil
		},
		SelectDB:   redisDatabase,
		ClientName: "metal-apiserver",
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create valkey client: %w", err)
	}

	return valkeyClient, nil
}
