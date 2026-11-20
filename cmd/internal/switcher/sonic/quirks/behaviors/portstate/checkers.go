package portstate

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// PortStateChecker tells whether a port is still bridged or routed before it gets reconfigured.
type PortStateChecker interface {
	Refresh(ctx context.Context) error
	// NotBridged returns the conditions that hold once the port is no member of any vlan
	NotBridged(interfaceName string) []db.Condition
	// NotRouted returns the conditions that hold once the port has no router interface
	NotRouted(interfaceName string) []db.Condition
	// BridgedIn returns the conditions that hold once the port is a member of the vlan
	BridgedIn(interfaceName, vlan string) []db.Condition
	// RoutedIn returns the conditions that hold once the port is a router interface in the vrf
	RoutedIn(interfaceName, vrf string) []db.Condition
	// HasAdminStatus returns the conditions that hold once the port has the admin status
	HasAdminStatus(interfaceName string, status types.PortStatus) []db.Condition
}

// kernelAdminStatus holds while the netdev of the port is administratively up or down.
func kernelAdminStatus(interfaceName string, status types.PortStatus) db.Condition {
	return db.NewCondition(fmt.Sprintf("kernel %s admin %s", interfaceName, status), func(context.Context) (bool, string, error) {
		nic, err := net.InterfaceByName(interfaceName)
		if err != nil {
			return false, "", err
		}
		up := nic.Flags&net.FlagUp != 0
		return up == (status == types.PortStatusUp), "flags " + nic.Flags.String(), nil
	})
}

// absent holds while exists reports false.
func absent(desc string, exists func(ctx context.Context) (bool, error)) db.Condition {
	return db.NewCondition(desc, func(ctx context.Context) (bool, string, error) {
		found, err := exists(ctx)
		return !found, "", err
	})
}

// present holds while exists reports true.
func present(desc string, exists func(ctx context.Context) (bool, error)) db.Condition {
	return db.NewCondition(desc, func(ctx context.Context) (bool, string, error) {
		found, err := exists(ctx)
		return found, "", err
	})
}

// sonicPortState checks each step SONiC takes to apply a port configuration:
//   - the STATE_DB, where intfmgrd and vlanmgrd report that the kernel side is set up,
//   - the ASIC_DB, where syncd reports the bridge port and router interface it created in the ASIC,
//   - the netdev in the kernel for the admin status.
type sonicPortState struct {
	db  *db.DB
	log *slog.Logger

	// vlans is cached to look up vlan memberships by key instead of scanning the STATE_DB.
	// metal-core does not create vlans with bridged members while applying a config.
	vlans []string

	bridgePortOidMap map[string]db.OID
	portOidMap       map[string]db.OID
	rifOidMap        map[string]db.OID
}

func (s *sonicPortState) Refresh(ctx context.Context) error {
	vlans, err := s.db.State.GetVlans(ctx)
	if err != nil {
		return fmt.Errorf("could not update vlans: %w", err)
	}
	s.vlans = vlans

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

func (s *sonicPortState) NotBridged(interfaceName string) []db.Condition {
	conds := []db.Condition{absent(fmt.Sprintf("STATE_DB VLAN_MEMBER_TABLE of %s absent", interfaceName), func(ctx context.Context) (bool, error) {
		return s.db.State.ExistVlanMember(ctx, interfaceName, s.vlans)
	})}
	if oid, ok := s.bridgePortOidMap[interfaceName]; ok {
		conds = append(conds, absent(fmt.Sprintf("ASIC_DB bridge port %s of %s absent", oid, interfaceName), func(ctx context.Context) (bool, error) {
			return s.db.Asic.ExistBridgePort(ctx, oid)
		}))
	}
	return conds
}

func (s *sonicPortState) NotRouted(interfaceName string) []db.Condition {
	conds := []db.Condition{absent(fmt.Sprintf("STATE_DB INTERFACE_TABLE|%s absent", interfaceName), func(ctx context.Context) (bool, error) {
		return s.db.State.ExistInterface(ctx, interfaceName)
	})}
	if oid, ok := s.rifOidMap[interfaceName]; ok {
		conds = append(conds, absent(fmt.Sprintf("ASIC_DB router interface %s of %s absent", oid, interfaceName), func(ctx context.Context) (bool, error) {
			return s.db.Asic.ExistRouterInterface(ctx, oid)
		}))
	}
	return conds
}

// BridgedIn looks up the bridge port of the port again, as it does not exist when the oid maps are refreshed.
func (s *sonicPortState) BridgedIn(interfaceName, vlan string) []db.Condition {
	return []db.Condition{
		present(fmt.Sprintf("STATE_DB VLAN_MEMBER_TABLE|%s|%s exists", vlan, interfaceName), func(ctx context.Context) (bool, error) {
			return s.db.State.ExistVlanMember(ctx, interfaceName, []string{vlan})
		}),
		present(fmt.Sprintf("ASIC_DB bridge port of %s exists", interfaceName), func(ctx context.Context) (bool, error) {
			bridgePorts, err := s.db.Asic.GetPortIdBridgePortMap(ctx)
			if err != nil {
				return false, err
			}
			_, ok := bridgePorts[s.portOidMap[interfaceName]]
			return ok, nil
		}),
	}
}

// RoutedIn checks the vrf in the STATE_DB, as the router interface of a previous vrf may still exist.
// The ASIC_DB does not tell the vrf of a router interface, so only its existence is checked there.
// The router interface is looked up again, as it does not exist when the oid maps are refreshed.
func (s *sonicPortState) RoutedIn(interfaceName, vrf string) []db.Condition {
	return []db.Condition{
		db.NewCondition(fmt.Sprintf("STATE_DB INTERFACE_TABLE|%s in %s", interfaceName, vrf), func(ctx context.Context) (bool, string, error) {
			current, exists, err := s.db.State.GetInterfaceVrf(ctx, interfaceName)
			if err != nil || !exists {
				return false, "absent", err
			}
			return current == vrf, fmt.Sprintf("in %q", current), nil
		}),
		present(fmt.Sprintf("ASIC_DB router interface of %s exists", interfaceName), func(ctx context.Context) (bool, error) {
			rifs, err := s.db.Counters.GetRifNameMap(ctx)
			if err != nil {
				return false, err
			}
			oid, ok := rifs[interfaceName]
			if !ok {
				return false, nil
			}
			return s.db.Asic.ExistRouterInterface(ctx, oid)
		}),
	}
}

func (s *sonicPortState) HasAdminStatus(interfaceName string, status types.PortStatus) []db.Condition {
	return []db.Condition{kernelAdminStatus(interfaceName, status)}
}
