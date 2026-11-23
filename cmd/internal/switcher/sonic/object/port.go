package object

import (
	"maps"
	"slices"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const (
	linkLocalOnly = "ipv6_use_link_local_only"
	vrfName       = "vrf_name"
)

// Port sets the mtu and the admin status of a port. Ports exist as long as the switch has them,
// so a Port is never deleted.
type Port struct {
	entry
	name        string
	adminStatus types.PortStatus
}

// NewPort sets the mtu, and the admin status unless it is empty.
func NewPort(name, mtu string, adminStatus types.PortStatus) Port {
	fields := db.Val{"mtu": mtu}
	if adminStatus != "" {
		fields["admin_status"] = string(adminStatus)
	}
	return Port{entry: entry{key: db.Key{"PORT", name}, fields: fields}, name: name, adminStatus: adminStatus}
}

// Owned are only the fields that are set, the other fields of a port are part of the switch configuration.
func (p Port) Owned() []string {
	return slices.Sorted(maps.Keys(p.fields))
}

// Writable only updates ports the switch has, a PORT entry is never created.
func (p Port) Writable(s *observe.Snapshot) Status {
	return exists("CONFIG_DB", s.Config, p.key)
}

// Applied waits for the admin status of the netdev, portmgrd applies the mtu along with it.
func (p Port) Applied(s *observe.Snapshot) Status {
	if p.adminStatus == "" {
		return Done
	}
	up, ok := s.AdminUp[p.name]
	if !ok {
		return check(false, "kernel %s exists", p.name)
	}
	return check(up == (p.adminStatus == types.PortStatusUp), "kernel %s admin %s", p.name, p.adminStatus)
}

func (p Port) Removed(*observe.Snapshot) Status {
	return Done
}

// RouterInterface makes a port a router interface in a vrf, the default vrf is "". It has no address,
// BGP peers over the IPv6 link-local address.
//
// intfmgrd creates the interface in the kernel, orchagent its router interface. orchagent can only remove the
// router interface once no route uses it anymore.
type RouterInterface struct {
	entry
	port, vrf string
}

func NewRouterInterface(port, vrf string) RouterInterface {
	fields := db.Val{linkLocalOnly: "enable"}
	if vrf != "" {
		fields[vrfName] = vrf
	}
	return RouterInterface{entry: entry{key: db.Key{"INTERFACE", port}, fields: fields}, port: port, vrf: vrf}
}

// Owned leaves out other fields of the interface, like its loopback action on AsterNOS.
func (i RouterInterface) Owned() []string {
	return []string{linkLocalOnly, vrfName}
}

func (i RouterInterface) Requires() []db.Key {
	if i.vrf == "" {
		return nil
	}
	return []db.Key{{"VRF", i.vrf}}
}

// Writable waits until the port is not bridged anymore, a port is either bridged or routed.
func (i RouterInterface) Writable(s *observe.Snapshot) Status {
	return notBridged(s, i.port)
}

// Replace is needed for another vrf. intfmgrd would move the interface, but orchagent keeps the router interface
// of the previous vrf as long as routes use it.
func (i RouterInterface) Replace(current db.Val) bool {
	return current[vrfName] != i.vrf
}

func (i RouterInterface) Applied(s *observe.Snapshot) Status {
	oid, ok := s.RifOIDs[i.port]
	return all(
		interfaceInVrf(s, i.port, i.vrf),
		check(ok && s.Asic.Has(routerInterfaceKey(oid)), "ASIC_DB router interface of %s exists", i.port),
	)
}

func (i RouterInterface) Removed(s *observe.Snapshot) Status {
	return notRouted(s, i.port)
}

// Removable waits until FRR no longer peers on the port, and orchagent removed the routes via the port.
//
// Routes that are withdrawn after the interface left its vrf may never be removed from it,
// so the interface must stay in its vrf until then.
func (i RouterInterface) Removable(s *observe.Snapshot, frr FRR) Status {
	return all(
		check(!frr.UsesPort(i.vrf, i.port), "FRR without %s in vrf %q applied", i.port, i.vrf),
		noRoutesVia(s, i.vrf, i.port),
	)
}

func (i RouterInterface) RoutesVrf() string {
	return i.vrf
}

// VlanMember makes a port an untagged member of a vlan.
//
// vlanmgrd adds the port to the bridge in the kernel, orchagent creates its bridge port.
type VlanMember struct {
	entry
	vlan, port string
}

func NewVlanMember(vlan, port string) VlanMember {
	return newVlanMember(vlan, port, db.Val{"tagging_mode": "untagged"})
}

func newVlanMember(vlan, port string, fields db.Val) VlanMember {
	return VlanMember{entry: entry{key: db.Key{"VLAN_MEMBER", vlan, port}, fields: fields}, vlan: vlan, port: port}
}

// Requires the vlan, which is part of the switch configuration for the PXE vlan.
func (m VlanMember) Requires() []db.Key {
	return []db.Key{{"VLAN", m.vlan}}
}

// Writable waits until the port is not routed anymore, a port is either bridged or routed.
func (m VlanMember) Writable(s *observe.Snapshot) Status {
	return notRouted(s, m.port)
}

func (m VlanMember) Applied(s *observe.Snapshot) Status {
	_, bridged := bridgePort(s, m.port)
	return all(
		exists("STATE_DB", s.State, db.Key{"VLAN_MEMBER_TABLE", m.vlan, m.port}),
		check(bridged, "ASIC_DB bridge port of %s exists", m.port),
	)
}

func (m VlanMember) Removed(s *observe.Snapshot) Status {
	oid, bridged := bridgePort(s, m.port)
	return all(
		absent("STATE_DB", s.State, db.Key{"VLAN_MEMBER_TABLE", m.vlan, m.port}),
		check(!bridged, "ASIC_DB bridge port %s of %s absent", oid, m.port),
	)
}

// notRouted is pending while intfmgrd or orchagent have the port set up as router interface.
func notRouted(s *observe.Snapshot, port string) Status {
	oid, ok := s.RifOIDs[port]
	return all(
		absent("STATE_DB", s.State, db.Key{"INTERFACE_TABLE", port}),
		check(!ok || !s.Asic.Has(routerInterfaceKey(oid)), "ASIC_DB router interface %s of %s absent", oid, port),
	)
}

// notBridged is pending while vlanmgrd or orchagent have the port set up as member of a vlan.
func notBridged(s *observe.Snapshot, port string) Status {
	var statuses []Status
	for _, key := range s.State.Keys(db.Key{"VLAN_MEMBER_TABLE"}) {
		if len(key) == 3 && key[2] == port {
			statuses = append(statuses, absent("STATE_DB", s.State, key))
		}
	}
	oid, bridged := bridgePort(s, port)
	statuses = append(statuses, check(!bridged, "ASIC_DB bridge port %s of %s absent", oid, port))
	return all(statuses...)
}

func routerInterfaceKey(oid db.OID) db.Key {
	return db.Key{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE", string(oid)}
}

// bridgePort returns the bridge port orchagent created for the port, if any.
func bridgePort(s *observe.Snapshot, port string) (db.OID, bool) {
	portOID, ok := s.PortOIDs[port]
	if !ok {
		return "", false
	}
	table := db.Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT"}
	for _, oid := range s.Asic.Items(table) {
		val, _ := s.Asic.Get(db.Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", oid})
		if db.OID(val["SAI_BRIDGE_PORT_ATTR_PORT_ID"]) == portOID {
			return db.OID(oid), true
		}
	}
	return "", false
}
