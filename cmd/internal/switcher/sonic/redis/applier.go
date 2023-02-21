package redis

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-cmp/cmp"
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
	c           *configDB
	log         *zap.SugaredLogger
	previousCfg *types.Conf
	s           *stateDB
}

func NewApplier(log *zap.SugaredLogger, cfg *Config) *Applier {
	configDB := cfg.Databases["CONFIG_DB"]
	stateDB := cfg.Databases["STATE_DB"]

	return &Applier{
		c:   newConfigDB(cfg.Instances[configDB.Instance].Addr, configDB.Id, configDB.Separator),
		log: log,
		s:   newStateDB(cfg.Instances[stateDB.Instance].Addr, stateDB.Id, stateDB.Separator),
	}
}

func (a *Applier) Apply(cfg *types.Conf) error {
	var errs []error

	// only process if changes are detected
	if a.previousCfg != nil {
		diff := cmp.Diff(a.previousCfg, cfg)
		if diff == "" {
			a.log.Infow("no changes on interfaces detected, nothing to do")
			return nil
		} else {
			a.log.Debugw("interface changes", "changes", diff)
		}
	}

	for _, interfaceName := range cfg.Ports.Underlay {
		if err := a.configureUnderlayPort(interfaceName); err != nil {
			errs = append(errs, err)
		}
	}

	for _, interfaceName := range cfg.Ports.Unprovisioned {
		if err := a.configureUnprovisionedPort(interfaceName); err != nil {
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
			if err := a.configureVrfNeighbor(interfaceName, vrfName); err != nil {
				errs = append(errs, err)
			}
		}
	}

	// config is only treated as applied if no errors are encountered
	if len(errs) == 0 {
		a.previousCfg = cfg
	}
	return errors.Join(errs...)
}

func (a *Applier) configureUnprovisionedPort(interfaceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureInterfaceIsNotVrfMember(ctx, interfaceName)
	if err != nil {
		return err
	}

	err = a.ensureLinkLocalOnlyIsDisabled(ctx, interfaceName)
	if err != nil {
		return err
	}

	return a.ensureInterfaceIsVlanMember(ctx, interfaceName, "Vlan4000")
}

func (a *Applier) configureFirewallPort(interfaceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureInterfaceIsNotVlanMember(ctx, interfaceName)
	if err != nil {
		return err
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureUnderlayPort(interfaceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureVrfNeighbor(interfaceName, vrf string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureInterfaceIsNotVlanMember(ctx, interfaceName)
	if err != nil {
		return err
	}

	return a.ensureInterfaceIsVrfMember(ctx, interfaceName, vrf)
}
