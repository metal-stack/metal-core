package quirks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

// PortStateChecker tells whether a port is still bridged or routed before it gets reconfigured.
type PortStateChecker interface {
	Refresh(ctx context.Context) error
	IsBridged(ctx context.Context, interfaceName string) (bool, error)
	IsRouted(ctx context.Context, interfaceName string) (bool, error)
}

// asicDBPortState checks the SAI objects in the ASIC_DB, i.e. waits until syncd removed the
// bridge port or router interface from the ASIC.
type asicDBPortState struct {
	db  *db.DB
	log *slog.Logger

	bridgePortOidMap map[string]db.OID
	portOidMap       map[string]db.OID
	rifOidMap        map[string]db.OID
}

func (s *asicDBPortState) Refresh(ctx context.Context) error {
	s.log.Debug("refresh oid maps")

	oidMap, err := s.db.Counters.GetPortNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update port to oid map: %w", err)
	}
	s.log.Debug("set port oid map", "map", oidMap)
	s.portOidMap = oidMap

	oidMap, err = s.db.Counters.GetRifNameMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update rif to oid ma: %w", err)
	}
	s.log.Debug("set rif oid map", "map", oidMap)
	s.rifOidMap = oidMap

	bridgePortMap, err := s.db.Asic.GetPortIdBridgePortMap(ctx)
	if err != nil {
		return fmt.Errorf("could not update bridge port to oid map: %w", err)
	}
	oidMap = make(map[string]db.OID, len(bridgePortMap))
	for port, oid := range s.portOidMap {
		if bridgePort, ok := bridgePortMap[oid]; ok {
			oidMap[port] = bridgePort
		}
	}
	s.log.Debug("set bridge port oid map", "map", oidMap)
	s.bridgePortOidMap = oidMap

	return nil
}

func (s *asicDBPortState) IsBridged(ctx context.Context, interfaceName string) (bool, error) {
	oid, ok := s.bridgePortOidMap[interfaceName]
	if !ok {
		return false, nil
	}
	return s.db.Asic.ExistBridgePort(ctx, oid)
}

func (s *asicDBPortState) IsRouted(ctx context.Context, interfaceName string) (bool, error) {
	oid, ok := s.rifOidMap[interfaceName]
	if !ok {
		return false, nil
	}
	return s.db.Asic.ExistRouterInterface(ctx, oid)
}

// stateDBPortState uses the STATE_DB to report port status
type stateDBPortState struct {
	db *db.DB
}

func (s *stateDBPortState) Refresh(_ context.Context) error {
	return nil
}

func (s *stateDBPortState) IsBridged(ctx context.Context, interfaceName string) (bool, error) {
	return s.db.State.ExistVlanMember(ctx, interfaceName)
}

func (s *stateDBPortState) IsRouted(ctx context.Context, interfaceName string) (bool, error) {
	return s.db.State.ExistInterface(ctx, interfaceName)
}
