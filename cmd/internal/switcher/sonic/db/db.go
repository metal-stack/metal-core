package db

import (
	"fmt"
	"os"
	"strings"

	"github.com/valkey-io/valkey-go"
)

type (
	Config struct {
		Databases map[string]Database `json:"DATABASES"`
		Instances map[string]Instance `json:"INSTANCES"`
	}

	Database struct {
		Id        int    `json:"id"`
		Instance  string `json:"instance"`
		Separator string `json:"separator"`
	}

	Instance struct {
		Hostname       string `json:"hostname"`
		UnixSocketPath string `json:"unix_socket_path"`
		PasswordPath   string `json:"password_path"`
	}

	DB struct {
		Appl     *ApplDB
		Asic     *AsicDB
		Config   *ConfigDB
		Counters *CountersDB
	}

	Scheme string
)

const (
	SchemeUnix  = Scheme("unix")
	SchemeRedis = Scheme("redis")
)

func New(cfg *Config, scheme Scheme) (*DB, error) {
	applDB := cfg.Databases["APPL_DB"]
	asicDB := cfg.Databases["ASIC_DB"]
	configDB := cfg.Databases["CONFIG_DB"]
	countersDB := cfg.Databases["COUNTERS_DB"]

	applClient, err := newRedisClient(cfg.Instances[applDB.Instance], applDB.Id, scheme)
	if err != nil {
		return nil, fmt.Errorf("could not create client for APPL_DB: %w", err)
	}

	asicClient, err := newRedisClient(cfg.Instances[asicDB.Instance], asicDB.Id, scheme)
	if err != nil {
		return nil, fmt.Errorf("could not create client for ASIC_DB: %w", err)
	}

	configClient, err := newRedisClient(cfg.Instances[configDB.Instance], configDB.Id, scheme)
	if err != nil {
		return nil, fmt.Errorf("could not create client for CONFIG_DB: %w", err)
	}

	countersClient, err := newRedisClient(cfg.Instances[countersDB.Instance], countersDB.Id, scheme)
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

func newRedisClient(redisInstance Instance, redisDatabase int, scheme Scheme) (valkey.Client, error) {
	var url string
	switch scheme {
	case SchemeUnix:
		url = fmt.Sprintf("%s://%s?db=%d", scheme, redisInstance.UnixSocketPath, redisDatabase)
	case SchemeRedis:
		url = fmt.Sprintf("%s://%s/%d", scheme, redisInstance.Hostname, redisDatabase)
	default:
		return nil, fmt.Errorf("invalid scheme %s for redis connection", scheme)
	}

	opt, err := valkey.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url %s: %w", url, err)
	}
	opt.ClientName = "metal-core"
	opt.DisableCache = true

	if redisInstance.PasswordPath != "" {
		passwd, err := os.ReadFile(redisInstance.PasswordPath)
		if err != nil {
			return nil, fmt.Errorf("could not read password from %s: %w", redisInstance.PasswordPath, err)
		}
		opt.Password = strings.TrimSpace(string(passwd))
	}

	valkeyClient, err := valkey.NewClient(opt)
	if err != nil {
		return nil, fmt.Errorf("unable to create valkey client: %w", err)
	}

	return valkeyClient, nil
}
