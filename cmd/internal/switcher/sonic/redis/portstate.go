package redis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

// PortStateSource selects the database that is used to check whether a port is still bridged or routed
// before it gets reconfigured.
type PortStateSource string

const (
	// PortStateSourceAsicDB checks the SAI objects in the ASIC_DB, i.e. waits until syncd removed the
	// bridge port or router interface from the ASIC.
	PortStateSourceAsicDB = PortStateSource("asic_db")
	// PortStateSourceStateDB checks the state published by vlanmgrd and intfmgrd in the STATE_DB, i.e. waits
	// until the vlan membership or router interface was removed from the kernel and handed to orchagent.
	PortStateSourceStateDB = PortStateSource("state_db")
)

type portStateChecker interface {
	refresh(ctx context.Context) error
	isBridged(ctx context.Context, interfaceName string) (bool, error)
	isRouted(ctx context.Context, interfaceName string) (bool, error)
}

func newPortStateChecker(log *slog.Logger, sonicDb *db.DB, source PortStateSource) (portStateChecker, error) {
	switch source {
	case PortStateSourceAsicDB:
		return &asicDBPortState{db: sonicDb, log: log}, nil
	case PortStateSourceStateDB:
		return &stateDBPortState{db: sonicDb}, nil
	default:
		return nil, fmt.Errorf("unknown port state source %q, must be one of %q or %q", source, PortStateSourceAsicDB, PortStateSourceStateDB)
	}
}

type asicDBPortState struct {
	db  *db.DB
	log *slog.Logger

	bridgePortOidMap map[string]db.OID
	portOidMap       map[string]db.OID
	rifOidMap        map[string]db.OID
}

func (s *asicDBPortState) refresh(ctx context.Context) error {
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

func (s *asicDBPortState) isBridged(ctx context.Context, interfaceName string) (bool, error) {
	oid, ok := s.bridgePortOidMap[interfaceName]
	if !ok {
		return false, nil
	}
	return s.db.Asic.ExistBridgePort(ctx, oid)
}

func (s *asicDBPortState) isRouted(ctx context.Context, interfaceName string) (bool, error) {
	oid, ok := s.rifOidMap[interfaceName]
	if !ok {
		return false, nil
	}
	return s.db.Asic.ExistRouterInterface(ctx, oid)
}

type stateDBPortState struct {
	db *db.DB
}

func (s *stateDBPortState) refresh(_ context.Context) error {
	return nil
}

func (s *stateDBPortState) isBridged(ctx context.Context, interfaceName string) (bool, error) {
	return s.db.State.ExistVlanMember(ctx, interfaceName)
}

func (s *stateDBPortState) isRouted(ctx context.Context, interfaceName string) (bool, error) {
	return s.db.State.ExistInterface(ctx, interfaceName)
}
