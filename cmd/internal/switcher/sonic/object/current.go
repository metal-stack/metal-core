package object

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
)

// Tables are the tables the objects are read from, and the ones SONiC reports to.
var Tables = observe.Tables{
	Config: []db.Key{
		{"PORT"}, {"INTERFACE"}, {"VLAN"}, {"VLAN_MEMBER"}, {"VLAN_INTERFACE"}, {"SUPPRESS_VLAN_NEIGH"},
		{"VRF"}, {"VXLAN_TUNNEL"}, {"VXLAN_TUNNEL_MAP"},
	},
	State: []db.Key{
		{"VRF_TABLE"}, {"VRF_OBJECT_TABLE"}, {"VLAN_TABLE"}, {"VLAN_MEMBER_TABLE"}, {"INTERFACE_TABLE"},
		{"SUPPRESS_VLAN_NEIGH_TABLE"},
	},
	Appl: []db.Key{{"VRF_TABLE"}, {"VXLAN_VRF_TABLE"}, {"VXLAN_TUNNEL_MAP_TABLE"}, {"INTF_TABLE"}},
	Asic: []db.Key{
		{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE"},
		{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT"},
	},
}

// VTEP returns the name of the vtep of the switch, which is part of the switch configuration.
func VTEP(s *observe.Snapshot) (string, error) {
	keys := s.Config.Keys(db.Key{"VXLAN_TUNNEL"})
	if len(keys) != 1 {
		return "", fmt.Errorf("could not find name of the vtep, found %d", len(keys))
	}
	return keys[0][1], nil
}

// Current returns the objects metal-core manages that are in the CONFIG_DB, and the ones SONiC still has set up
// after they were deleted from it. metal-core manages
//   - the tenant vrfs named Vrf<vni>, and their vlans, whose ids it takes from vlan.VlanIDMin to vlan.VlanIDMax,
//   - the router interfaces and vlan memberships of the given ports.
//
// An object that is only set up in SONiC has the fields SONiC reports, they tell what it requires.
func Current(s *observe.Snapshot, ports []string) []Object {
	c := &collector{seen: map[string]bool{}}

	for key, val := range entries(s.Config, "VRF") {
		if isTenantVrf(key[1]) {
			c.add(NewVrf(key[1], val))
		}
	}
	for _, table := range []db.Key{{"VRF_TABLE"}, {"VRF_OBJECT_TABLE"}} {
		for key := range entries(s.State, table...) {
			if isTenantVrf(key[1]) {
				c.add(NewVrf(key[1], nil))
			}
		}
	}
	for key := range entries(s.Appl, "VRF_TABLE") {
		if isTenantVrf(key[1]) {
			c.add(NewVrf(key[1], nil))
		}
	}

	for key, val := range entries(s.Config, "VLAN") {
		if isTenantVlan(key[1]) {
			c.add(newVlan(key[1], val))
		}
	}
	for key := range entries(s.State, "VLAN_TABLE") {
		if isTenantVlan(key[1]) {
			c.add(newVlan(key[1], nil))
		}
	}

	for key, val := range entries(s.Config, "SUPPRESS_VLAN_NEIGH") {
		if isTenantVlan(key[1]) {
			c.add(newSuppressVlanNeigh(key[1], val))
		}
	}
	for key := range entries(s.State, "SUPPRESS_VLAN_NEIGH_TABLE") {
		if isTenantVlan(key[1]) {
			c.add(newSuppressVlanNeigh(key[1], nil))
		}
	}

	for key, val := range entries(s.Config, "VLAN_INTERFACE") {
		if isTenantVlan(key[1]) {
			c.add(NewVlanInterface(key[1], val[vrfName]))
		}
	}
	for key, val := range entries(s.State, "INTERFACE_TABLE") {
		if isTenantVlan(key[1]) {
			c.add(NewVlanInterface(key[1], val["vrf"]))
		}
	}
	for key, val := range entries(s.Appl, "INTF_TABLE") {
		if isTenantVlan(key[1]) {
			c.add(NewVlanInterface(key[1], val[vrfName]))
		}
	}

	for key, val := range members(s.Config, "VXLAN_TUNNEL_MAP") {
		if isTenantVlan(val["vlan"]) {
			c.add(newVxlanTunnelMap(key[1], key[2], val))
		}
	}
	for key, val := range members(s.Appl, "VXLAN_TUNNEL_MAP_TABLE") {
		if isTenantVlan(val["vlan"]) {
			c.add(newVxlanTunnelMap(key[1], key[2], val))
		}
	}

	for key, val := range entries(s.Config, "INTERFACE") {
		if slices.Contains(ports, key[1]) {
			c.add(NewRouterInterface(key[1], val[vrfName]))
		}
	}
	for key, val := range entries(s.State, "INTERFACE_TABLE") {
		if slices.Contains(ports, key[1]) {
			c.add(NewRouterInterface(key[1], val["vrf"]))
		}
	}

	for key, val := range members(s.Config, "VLAN_MEMBER") {
		if slices.Contains(ports, key[2]) {
			c.add(newVlanMember(key[1], key[2], val))
		}
	}
	for key := range members(s.State, "VLAN_MEMBER_TABLE") {
		if slices.Contains(ports, key[2]) {
			c.add(newVlanMember(key[1], key[2], nil))
		}
	}

	return c.objects
}

type collector struct {
	objects []Object
	seen    map[string]bool
}

// add adds the object unless one with its key was added before, the CONFIG_DB entry goes first.
func (c *collector) add(o Object) {
	k := strings.Join(o.Key(), "|")
	if c.seen[k] {
		return
	}
	c.seen[k] = true
	c.objects = append(c.objects, o)
}

// entries iterates over the entries of the table with keys of two parts. Longer keys hold the addresses of
// an interface or members, they are not objects of the table.
func entries(d db.Dump, table ...string) func(yield func(db.Key, db.Val) bool) {
	return withParts(d, 2, table...)
}

// members iterates over the entries of a table whose keys have three parts, like members of a vlan.
func members(d db.Dump, table ...string) func(yield func(db.Key, db.Val) bool) {
	return withParts(d, 3, table...)
}

func withParts(d db.Dump, parts int, table ...string) func(yield func(db.Key, db.Val) bool) {
	return func(yield func(db.Key, db.Val) bool) {
		for _, key := range d.Keys(table) {
			if len(key) != parts {
				continue
			}
			val, _ := d.Get(key)
			if !yield(key, val) {
				return
			}
		}
	}
}

func isTenantVrf(name string) bool {
	vni, ok := strings.CutPrefix(name, "Vrf")
	if !ok {
		return false
	}
	_, err := strconv.ParseUint(vni, 10, 32)
	return err == nil
}

func isTenantVlan(name string) bool {
	id, ok := strings.CutPrefix(name, "Vlan")
	if !ok {
		return false
	}
	vid, err := strconv.ParseUint(id, 10, 16)
	return err == nil && uint16(vid) >= vlan.VlanIDMin && uint16(vid) <= vlan.VlanIDMax
}
