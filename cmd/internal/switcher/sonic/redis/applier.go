package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type (
	Applier struct {
		db          *db.DB
		log         *slog.Logger
		previousCfg *types.Conf

		bridgePortOidMap map[string]db.OID
		portOidMap       map[string]db.OID
		rifOidMap        map[string]db.OID
		asic
	}

	asic interface {
		EnsureNotRouted(ctx context.Context, interfaceName string) error
		EnsureNotBridged(ctx context.Context, interfaceName string) error
	}

	testASIC struct {
		db *db.DB
	}
)

func NewApplier(log *slog.Logger, db *db.DB) *Applier {
	applier := &Applier{
		db:  db,
		log: log,
	}
	applier.asic = applier

	return applier
}

func NewTestApplier(log *slog.Logger, db *db.DB) *Applier {
	asic := &testASIC{
		db: db,
	}

	return &Applier{
		db:   db,
		log:  log,
		asic: asic,
	}
}

func (a *Applier) Apply(ctx context.Context, cfg *types.Conf) error {
	var (
		errs    []error
		changed bool
	)

	diff := cmp.Diff(a.previousCfg, cfg)
	if diff != "" {
		changed = true
		a.log.Debug("interface changes", "changes", diff)
	}

	currentPorts, err := a.db.Config.GetPorts(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	for _, port := range currentPorts {
		cfgPort, found := lo.Find(cfg.Ports, func(p types.Port) bool {
			return p.Name == port.Name
		})
		if !found {
			continue
		}

		if types.PortStatus(port.AdminStatus) != cfgPort.AdminStatus {
			changed = true
			break
		}
	}

	if !changed {
		a.log.Info("no changes on interfaces detected, nothing to do")
		return nil
	}

	if err := a.refreshOidMaps(ctx); err != nil {
		return err
	}

	a.log.Debug("configure ports", "ports", cfg.Ports)
	for _, port := range cfg.Ports {
		if err := a.configurePort(ctx, port); err != nil {
			errs = append(errs, err)
		}
	}

	for _, vrf := range cfg.Vrfs {
		if err := a.configureVrf(ctx, vrf); err != nil {
			errs = append(errs, err)
		}
	}

	err = a.cleanupVrfs(ctx, cfg)
	if err != nil {
		errs = append(errs, err)
	}

	// config is only treated as applied if no errors are encountered
	if len(errs) == 0 {
		a.previousCfg = cfg
	}
	return errors.Join(errs...)
}

func (a *Applier) GetPorts(ctx context.Context) ([]*db.Port, error) {
	return a.db.Config.GetPorts(ctx)
}

func (a *Applier) refreshOidMaps(ctx context.Context) error {
	a.log.Debug("refresh oid maps")

	oidMap, err := a.db.Counters.GetPortNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update port to oid map: %w", err)
	}
	a.log.Debug("set port oid map", "map", oidMap)
	a.portOidMap = oidMap

	oidMap, err = a.db.Counters.GetRifNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update rif to oid ma: %w", err)
	}
	a.log.Debug("set rif oid map", "map", oidMap)
	a.rifOidMap = oidMap

	bridgePortMap, err := a.db.Asic.GetPortIdBridgePortMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update bridge port to oid map: %w", err)
	}
	oidMap = make(map[string]db.OID, len(bridgePortMap))
	for port, oid := range a.portOidMap {
		if bridgePort, ok := bridgePortMap[oid]; ok {
			oidMap[port] = bridgePort
		}
	}
	a.log.Debug("set bridge port oid map", "map", oidMap)
	a.bridgePortOidMap = oidMap

	return nil
}
