package redis

import (
	"context"
	"errors"
	"time"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"go.uber.org/zap"
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
	Addr string `json:"unix_socket_path"`
}

type Applier struct {
	c   *configdb
	log *zap.SugaredLogger
}

func NewApplier(log *zap.SugaredLogger, cfg *Config) *Applier {
	db := cfg.Databases["CONFIG_DB"]

	return &Applier{
		c:   newConfigdb(cfg.Instances[db.Instance].Addr, db.Id, db.Separator),
		log: log,
	}
}

func (a *Applier) Apply(cfg *types.Conf) error {
	var errs []error

	for _, interfaceName := range cfg.Ports.Unprovisioned {
		if err := a.addInterfaceToVlan(interfaceName, "Vlan4000"); err != nil {
			errs = append(errs, err)
		}
	}

	for interfaceName := range cfg.Ports.Firewalls {
		if err := a.configureFirewallPort(interfaceName); err != nil {
			errs = append(errs, err)
		}
	}

	for vrfName, vrf := range cfg.Ports.Vrfs {
		for _, interfaceName := range vrf.Neighbors {
			if err := a.addInterfaceToVrf(interfaceName, vrfName); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

func (a *Applier) configureFirewallPort(interfaceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Firewalls have to be removed from the VLAN and specify no VRF
	return a.removeInterfaceFromVlan(ctx, interfaceName)
}
