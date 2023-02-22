package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"go.uber.org/zap"
)

type Applier struct {
	db          *db.DB
	log         *zap.SugaredLogger
	previousCfg *types.Conf
}

func NewApplier(log *zap.SugaredLogger, cfg *db.Config) *Applier {
	return &Applier{
		db:  db.New(cfg),
		log: log,
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

	err := a.ensureNotRouted(ctx, interfaceName)
	if err != nil {
		return err
	}

	if err := a.ensurePortMTU(ctx, interfaceName, "9000", true); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
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

	if err := a.ensurePortMTU(ctx, interfaceName, "9216", true); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureUnderlayPort(interfaceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.ensurePortMTU(ctx, interfaceName, "9216", false); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}
	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureVrfNeighbor(interfaceName, vrf string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureInterfaceIsNotVlanMember(ctx, interfaceName)
	if err != nil {
		return err
	}

	err = a.ensureInterfaceIsVrfMember(ctx, interfaceName, vrf)
	if err != nil {
		return err
	}

	if err := a.ensurePortMTU(ctx, interfaceName, "9000", true); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}
