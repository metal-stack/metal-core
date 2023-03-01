package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/redis/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type Applier struct {
	db          *db.DB
	log         *zap.SugaredLogger
	previousCfg *types.Conf

	portOidMap map[string]db.OID
}

type redisLogger struct {
	log *zap.SugaredLogger
}

func (l *redisLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	l.log.Infof(format, v...)
}

func NewApplier(log *zap.SugaredLogger, cfg *db.Config) *Applier {
	redis.SetLogger(&redisLogger{log: log})
	return &Applier{
		db:  db.New(cfg),
		log: log,
	}
}

func (a *Applier) Apply(cfg *types.Conf) (bool, error) {
	var errs []error

	// only process if changes are detected
	if a.previousCfg != nil {
		diff := cmp.Diff(a.previousCfg, cfg)
		if diff == "" {
			a.log.Infow("no changes on interfaces detected, nothing to do")
			return false, nil
		} else {
			a.log.Debugw("interface changes", "changes", diff)
		}
	}

	if err := a.refreshOidMaps(); err != nil {
		return true, err
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
		if err := a.configureVrf(vrfName, vrf); err != nil {
			errs = append(errs, err)
		}
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
	return true, errors.Join(errs...)
}

func (a *Applier) refreshOidMaps() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	oidMap, err := a.db.Counters.GetPortNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update port to oid map: %w", err)
	}
	a.portOidMap = oidMap

	return nil
}

func (a *Applier) configureUnprovisionedPort(interfaceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureNotRouted(ctx, interfaceName)
	if err != nil {
		return err
	}

	if err := a.ensurePortConfiguration(ctx, interfaceName, "9000", true); err != nil {
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

	if err := a.ensurePortConfiguration(ctx, interfaceName, "9216", true); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureUnderlayPort(interfaceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.ensurePortConfiguration(ctx, interfaceName, "9216", false); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}
	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureVrfNeighbor(interfaceName, vrfName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := a.ensureInterfaceIsNotVlanMember(ctx, interfaceName)
	if err != nil {
		return err
	}

	err = a.ensureInterfaceIsVrfMember(ctx, interfaceName, vrfName)
	if err != nil {
		return err
	}

	if err := a.ensurePortConfiguration(ctx, interfaceName, "9000", true); err != nil {
		return fmt.Errorf("failed to update Port info for interface %s: %w", interfaceName, err)
	}

	return a.ensureLinkLocalOnlyIsEnabled(ctx, interfaceName)
}

func (a *Applier) configureVrf(vrfName string, vrf *types.Vrf) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exist, err := a.db.Config.ExistVrf(ctx, vrfName)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVrf(ctx, vrfName); err != nil {
			return fmt.Errorf("could not create vrf %s: %w", vrfName, err)
		}
	}

	exist, err = a.db.Config.ExistVlan(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVlan(ctx, vrf.VLANID); err != nil {
			return fmt.Errorf("could not create vlan %d: %w", vrf.VLANID, err)
		}
	}

	exist, err = a.db.Config.ExistVlanInterface(ctx, vrf.VLANID)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVlanInterface(ctx, vrf.VLANID, vrfName); err != nil {
			return fmt.Errorf("could not create vlan interface for vlan %d: %w", vrf.VLANID, err)
		}
	}

	exist, err = a.db.Config.ExistVxlanTunnelMap(ctx, vrf.VLANID, vrf.VNI)
	if err != nil {
		return err
	}
	if !exist {
		if err := a.db.Config.CreateVxlanTunnelMap(ctx, vrf.VLANID, vrf.VNI); err != nil {
			return fmt.Errorf("could not create vxlan tunnel between vlan %d and vni %d: %w", vrf.VLANID, vrf.VNI, err)
		}
	}

	return nil
}
