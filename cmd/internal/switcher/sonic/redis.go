package sonic

import (
	"errors"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/configdb"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"go.uber.org/zap"
)

type sonicDatabasesConfig struct {
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

type redisApplier struct {
	c configdb.ConfigDB
}

func NewRedisApplier(log *zap.SugaredLogger, cfg *sonicDatabasesConfig) *redisApplier {
	db := cfg.Databases["CONFIG_DB"]

	return &redisApplier{
		c: configdb.New(log, &configdb.Options{
			Addr:      cfg.Instances[db.Instance].Addr,
			Id:        db.Id,
			Separator: db.Separator,
		}),
	}
}

func (a *redisApplier) apply(cfg *types.Conf) error {
	var (
		errs             []error
		interfaceConfigs []configdb.InterfaceConfiguration
	)

	for _, p := range cfg.Ports.Unprovisioned {
		interfaceConfigs = append(interfaceConfigs, configdb.InterfaceConfiguration{Name: p, Vlan: &configdb.Vlan{Name: "Vlan4000"}})
	}

	for vrfName, vrf := range cfg.Ports.Vrfs {
		for _, p := range vrf.Neighbors {
			interfaceConfigs = append(interfaceConfigs, configdb.InterfaceConfiguration{Name: p, Vrf: &configdb.Vrf{Name: vrfName}})
		}
	}

	for _, c := range interfaceConfigs {
		err := a.c.ConfigureInterface(c)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
