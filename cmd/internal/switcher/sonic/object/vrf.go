package object

import (
	"fmt"
	"maps"
	"strconv"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
)

// VrfName is the name of the tenant vrf with the vni.
func VrfName(vni uint32) string {
	return fmt.Sprintf("Vrf%d", vni)
}

// VlanName is the name of the vlan with the id.
func VlanName(id uint16) string {
	return fmt.Sprintf("Vlan%d", id)
}

// Vrf is a tenant vrf. Its fields depend on the NOS, see the VrfEntries quirk.
//
// vrfmgrd creates the vrf in the kernel, then orchagent creates its virtual router. orchagent can only remove the
// virtual router once no route and no router interface references it anymore.
type Vrf struct {
	entry
	name string
}

func NewVrf(name string, fields db.Val) Vrf {
	return Vrf{entry: entry{key: db.Key{"VRF", name}, fields: fields}, name: name}
}

func (v Vrf) Applied(s *observe.Snapshot) Status {
	return all(
		exists("STATE_DB", s.State, db.Key{"VRF_TABLE", v.name}),
		exists("STATE_DB", s.State, db.Key{"VRF_OBJECT_TABLE", v.name}),
	)
}

func (v Vrf) Removed(s *observe.Snapshot) Status {
	statuses := []Status{
		absent("STATE_DB", s.State, db.Key{"VRF_TABLE", v.name}),
		absent("STATE_DB", s.State, db.Key{"VRF_OBJECT_TABLE", v.name}),
		absent("APPL_DB", s.Appl, db.Key{"VRF_TABLE", v.name}),
	}
	// the binding of the vni to the vrf, set up by vxlanmgrd
	for _, item := range s.Appl.Items(db.Key{"VXLAN_VRF_TABLE"}) {
		key := db.Key{"VXLAN_VRF_TABLE", item}
		if val, _ := s.Appl.Get(key); val["vrf"] == v.name {
			statuses = append(statuses, absent("APPL_DB", s.Appl, key))
		}
	}
	return all(statuses...)
}

// Removable waits until FRR no longer has the vrf and orchagent removed its routes.
func (v Vrf) Removable(s *observe.Snapshot, frr FRR) Status {
	return all(
		check(!frr.UsesVrf(v.name), "FRR without %s applied", v.name),
		noRoutesIn(s, v.name),
	)
}

func (v Vrf) RoutesVrf() string {
	return v.name
}

// Vlan is the vlan of the l3 vni of a tenant vrf.
type Vlan struct {
	entry
	name string
}

func NewVlan(id uint16) Vlan {
	return newVlan(VlanName(id), db.Val{"vlanid": strconv.FormatUint(uint64(id), 10)})
}

func newVlan(name string, fields db.Val) Vlan {
	return Vlan{entry: entry{key: db.Key{"VLAN", name}, fields: fields}, name: name}
}

func (v Vlan) Applied(s *observe.Snapshot) Status {
	return exists("STATE_DB", s.State, db.Key{"VLAN_TABLE", v.name})
}

func (v Vlan) Removed(s *observe.Snapshot) Status {
	return absent("STATE_DB", s.State, db.Key{"VLAN_TABLE", v.name})
}

// SuppressVlanNeigh turns on ARP and ND suppression in the vlan.
type SuppressVlanNeigh struct {
	entry
	vlan string
}

func NewSuppressVlanNeigh(vlan string) SuppressVlanNeigh {
	return newSuppressVlanNeigh(vlan, db.Val{"suppress": "on"})
}

func newSuppressVlanNeigh(vlan string, fields db.Val) SuppressVlanNeigh {
	return SuppressVlanNeigh{entry: entry{key: db.Key{"SUPPRESS_VLAN_NEIGH", vlan}, fields: fields}, vlan: vlan}
}

func (n SuppressVlanNeigh) Requires() []db.Key {
	return []db.Key{{"VLAN", n.vlan}}
}

func (n SuppressVlanNeigh) Applied(s *observe.Snapshot) Status {
	return exists("STATE_DB", s.State, db.Key{"SUPPRESS_VLAN_NEIGH_TABLE", n.vlan})
}

func (n SuppressVlanNeigh) Removed(s *observe.Snapshot) Status {
	return absent("STATE_DB", s.State, db.Key{"SUPPRESS_VLAN_NEIGH_TABLE", n.vlan})
}

// VlanInterface binds the vlan to the vrf. intfmgrd creates the interface, orchagent its router interface.
type VlanInterface struct {
	entry
	vlan, vrf string
}

func NewVlanInterface(vlan, vrf string) VlanInterface {
	return VlanInterface{entry: entry{key: db.Key{"VLAN_INTERFACE", vlan}, fields: db.Val{"vrf_name": vrf}}, vlan: vlan, vrf: vrf}
}

func (i VlanInterface) Requires() []db.Key {
	return []db.Key{{"VLAN", i.vlan}, {"VRF", i.vrf}}
}

// Replace is needed for another vrf, as intfmgrd does not move an interface between vrfs.
func (i VlanInterface) Replace(current db.Val) bool {
	return current["vrf_name"] != i.vrf
}

func (i VlanInterface) Applied(s *observe.Snapshot) Status {
	return interfaceInVrf(s, i.vlan, i.vrf)
}

func (i VlanInterface) Removed(s *observe.Snapshot) Status {
	return all(
		absent("STATE_DB", s.State, db.Key{"INTERFACE_TABLE", i.vlan}),
		absent("APPL_DB", s.Appl, db.Key{"INTF_TABLE", i.vlan}),
	)
}

// Removable waits until FRR no longer has the vrf and orchagent removed the routes via the vlan. Routes from
// other VTEPs are routed via the vlan, so it is a router interface like the port of a machine.
func (i VlanInterface) Removable(s *observe.Snapshot, frr FRR) Status {
	return all(
		check(!frr.UsesVrf(i.vrf), "FRR without %s applied", i.vrf),
		noRoutesVia(s, i.vrf, i.vlan),
	)
}

func (i VlanInterface) RoutesVrf() string {
	return i.vrf
}

// VxlanTunnelMap maps the vni of a tenant vrf to its vlan.
type VxlanTunnelMap struct {
	entry
	vtep, name, vlan, vni string
}

func NewVxlanTunnelMap(vtep, vlan string, vni uint32) VxlanTunnelMap {
	v := strconv.FormatUint(uint64(vni), 10)
	return newVxlanTunnelMap(vtep, "map_"+v+"_"+vlan, db.Val{"vlan": vlan, "vni": v})
}

func newVxlanTunnelMap(vtep, name string, fields db.Val) VxlanTunnelMap {
	return VxlanTunnelMap{
		entry: entry{key: db.Key{"VXLAN_TUNNEL_MAP", vtep, name}, fields: fields},
		vtep:  vtep,
		name:  name,
		vlan:  fields["vlan"],
		vni:   fields["vni"],
	}
}

func (m VxlanTunnelMap) Requires() []db.Key {
	return []db.Key{{"VLAN", m.vlan}, {"VRF", m.vrf()}}
}

// Replace is needed for another vlan or vni, as vxlanmgrd does not update a map.
func (m VxlanTunnelMap) Replace(current db.Val) bool {
	return !maps.Equal(current, m.fields)
}

func (m VxlanTunnelMap) Applied(s *observe.Snapshot) Status {
	return exists("APPL_DB", s.Appl, db.Key{"VXLAN_TUNNEL_MAP_TABLE", m.vtep, m.name})
}

func (m VxlanTunnelMap) Removed(s *observe.Snapshot) Status {
	return absent("APPL_DB", s.Appl, db.Key{"VXLAN_TUNNEL_MAP_TABLE", m.vtep, m.name})
}

// Removable waits until FRR no longer has the vrf and orchagent removed its routes, the ones from other VTEPs
// use the vni.
func (m VxlanTunnelMap) Removable(s *observe.Snapshot, frr FRR) Status {
	return all(
		check(!frr.UsesVrf(m.vrf()), "FRR without %s applied", m.vrf()),
		noRoutesIn(s, m.vrf()),
	)
}

func (m VxlanTunnelMap) RoutesVrf() string {
	return m.vrf()
}

func (m VxlanTunnelMap) vrf() string {
	return "Vrf" + m.vni
}

// interfaceInVrf is pending until intfmgrd set up the interface in the vrf. The default vrf is "".
func interfaceInVrf(s *observe.Snapshot, name, vrf string) Status {
	key := db.Key{"INTERFACE_TABLE", name}
	val, ok := s.State.Get(key)
	if !ok {
		return check(false, "STATE_DB %s exists", s.State.String(key))
	}
	return check(val["vrf"] == vrf, "STATE_DB %s in vrf %q (is in %q)", s.State.String(key), vrf, val["vrf"])
}
