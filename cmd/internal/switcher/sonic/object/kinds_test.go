package object

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/sonictest"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func TestVrf(t *testing.T) {
	vrf := NewVrf("Vrf20", db.Val{"fallback": "false", "vni": "20"})

	s := read(t, leaf02(), "Vrf20")
	requireDone(t, vrf.Applied(s))
	requirePending(t, vrf.Removed(s), "APPL_DB VXLAN_VRF_TABLE:vtep:evpn_map_20_Vrf20 absent")
	requirePending(t, vrf.Removable(s, frr{vrfs: []string{"Vrf20"}}), "FRR without Vrf20 applied")
	requirePending(t, vrf.Removable(s, frr{}), "APPL_DB no routes in Vrf20 (ROUTE_TABLE:Vrf20:10.128.0.1)")

	s = read(t, without(leaf02(), "STATE_DB", "VRF_OBJECT_TABLE|Vrf20"))
	requirePending(t, vrf.Applied(s), "STATE_DB VRF_OBJECT_TABLE|Vrf20 exists")
	requirePending(t, vrf.Removable(s, frr{}), "routes of Vrf20 read")

	// after FRR removed the vrf, only the connected link-local route of the port is left, orchagent never programmed it
	s = read(t, without(leaf02(), "APPL_DB", "ROUTE_TABLE:Vrf20:10.128.0.1"), "Vrf20")
	requireDone(t, vrf.Removable(s, frr{}))

	s = read(t, sonictest.Data{})
	requireDone(t, vrf.Removed(s))
}

func TestVlanEntries(t *testing.T) {
	s := read(t, leaf02())
	for _, o := range []Object{
		NewVlan(1002),
		NewSuppressVlanNeigh("Vlan1002"),
		NewVlanInterface("Vlan1002", "Vrf20"),
		NewVxlanTunnelMap("vtep", "Vlan1002", 20),
	} {
		requireDone(t, o.Applied(s))
		require.False(t, o.Removed(s).Done(), "%v", o.Key())
	}

	// routes from other VTEPs are routed via the vlan and the vni
	evpn := leaf02()
	evpn["APPL_DB"]["ROUTE_TABLE"] = test.StringMap{
		"Vrf20": test.StringMap{"10.130.0.0/24": test.StringMap{"ifname": "Vlan1002", "nexthop": "10.0.0.11", "vni_label": "20"}},
	}
	es := read(t, evpn, "Vrf20")
	requirePending(t, NewVlanInterface("Vlan1002", "Vrf20").Removable(es, frr{}), "APPL_DB no routes via Vlan1002 in Vrf20 (ROUTE_TABLE:Vrf20:10.130.0.0/24)")
	requirePending(t, NewVxlanTunnelMap("vtep", "Vlan1002", 20).Removable(es, frr{}), "APPL_DB no routes in Vrf20 (ROUTE_TABLE:Vrf20:10.130.0.0/24)")
	requirePending(t, NewVlanInterface("Vlan1002", "Vrf20").Removable(es, frr{vrfs: []string{"Vrf20"}}), "FRR without Vrf20 applied")
	requirePending(t, NewVxlanTunnelMap("vtep", "Vlan1002", 20).Removable(es, frr{vrfs: []string{"Vrf20"}}), "FRR without Vrf20 applied")
	es = read(t, without(evpn, "APPL_DB", "ROUTE_TABLE:Vrf20:10.130.0.0/24"), "Vrf20")
	requireDone(t, NewVlanInterface("Vlan1002", "Vrf20").Removable(es, frr{}))
	requireDone(t, NewVxlanTunnelMap("vtep", "Vlan1002", 20).Removable(es, frr{}))

	requirePending(t, NewVlanInterface("Vlan1002", "Vrf30").Applied(s), `STATE_DB INTERFACE_TABLE|Vlan1002 in vrf "Vrf30" (is in "Vrf20")`)
	require.True(t, NewVlanInterface("Vlan1002", "Vrf30").Replace(db.Val{"vrf_name": "Vrf20"}))
	require.Equal(t, []db.Key{{"VLAN", "Vlan1002"}, {"VRF", "Vrf20"}}, NewVxlanTunnelMap("vtep", "Vlan1002", 20).Requires())

	s = read(t, sonictest.Data{})
	for _, o := range []Object{
		NewVlan(1002),
		NewSuppressVlanNeigh("Vlan1002"),
		NewVlanInterface("Vlan1002", "Vrf20"),
		NewVxlanTunnelMap("vtep", "Vlan1002", 20),
	} {
		requireDone(t, o.Removed(s))
	}
}

func TestRouterInterface(t *testing.T) {
	ri := NewRouterInterface("Ethernet3", "Vrf20")

	s := read(t, leaf02(), "Vrf20")
	requireDone(t, ri.Applied(s))
	requirePending(t, ri.Removed(s), "STATE_DB INTERFACE_TABLE|Ethernet3 absent")
	requirePending(t, ri.Removable(s, frr{ports: []string{"Vrf20|Ethernet3"}}), `FRR without Ethernet3 in vrf "Vrf20" applied`)
	requirePending(t, ri.Removable(s, frr{}), "APPL_DB no routes via Ethernet3 in Vrf20 (ROUTE_TABLE:Vrf20:10.128.0.1)")

	// the connected link-local route of the port does not keep it from being removed
	s = read(t, without(leaf02(), "APPL_DB", "ROUTE_TABLE:Vrf20:10.128.0.1"), "Vrf20")
	requireDone(t, ri.Removable(s, frr{}))

	require.Equal(t, db.Val{"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf20"}, ri.Fields())
	require.Equal(t, db.Val{"ipv6_use_link_local_only": "enable"}, NewRouterInterface("Ethernet120", "").Fields())
	require.True(t, ri.Replace(db.Val{"ipv6_use_link_local_only": "enable"}), "a firewall port moves into a vrf")
	require.False(t, ri.Replace(db.Val{"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf20"}))
	require.Nil(t, NewRouterInterface("Ethernet120", "").Requires())
}

// leaf03 (AsterNOS) after the vrf teardown race: intfmgrd removed the interface, but orchagent kept its router interface.
func TestRouterInterface_LeftInASIC(t *testing.T) {
	s := read(t, without(leaf02(), "STATE_DB", "INTERFACE_TABLE|Ethernet3"))

	requirePending(t, NewRouterInterface("Ethernet3", "Vrf20").Removed(s), "ASIC_DB router interface oid:0x6000000000dab of Ethernet3 absent")
	requirePending(t, NewVlanMember("Vlan4000", "Ethernet3").Writable(s), "ASIC_DB router interface oid:0x6000000000dab of Ethernet3 absent")
}

// leaf03 Ethernet1 after it was moved into Vrf46 while orchagent still had the router interface of Vrf20.
func TestRouterInterface_InPreviousVrf(t *testing.T) {
	s := read(t, leaf02())

	requirePending(t, NewRouterInterface("Ethernet3", "Vrf46").Applied(s), `STATE_DB INTERFACE_TABLE|Ethernet3 in vrf "Vrf46" (is in "Vrf20")`)
}

func TestRouterInterface_RoutesViaNextHopGroup(t *testing.T) {
	data := leaf02()
	data["APPL_DB"]["ROUTE_TABLE"] = test.StringMap{
		"Vrf20": test.StringMap{"10.130.0.0/24": test.StringMap{"nexthop_group": "7"}},
	}
	data["APPL_DB"]["NEXTHOP_GROUP_TABLE"] = test.StringMap{
		"7": test.StringMap{"ifname": "Ethernet2,Ethernet3", "nexthop": "fe80::1,fe80::2"},
	}
	s := read(t, data, "Vrf20")

	requirePending(t, NewRouterInterface("Ethernet3", "Vrf20").Removable(s, frr{}), "ROUTE_TABLE:Vrf20:10.130.0.0/24")
}

func TestVlanMember(t *testing.T) {
	s := read(t, leaf02())

	m := NewVlanMember("Vlan4000", "Ethernet0")
	requireDone(t, m.Applied(s))
	requireDone(t, m.Writable(s))
	requirePending(t, m.Removed(s), "ASIC_DB bridge port oid:0x3a000000000daa of Ethernet0 absent")
	requirePending(t, NewRouterInterface("Ethernet0", "").Writable(s), "STATE_DB VLAN_MEMBER_TABLE|Vlan4000|Ethernet0 absent")

	requirePending(t, NewVlanMember("Vlan4000", "Ethernet3").Writable(s), "STATE_DB INTERFACE_TABLE|Ethernet3 absent")
	requirePending(t, NewVlanMember("Vlan4000", "Ethernet3").Applied(s), "ASIC_DB bridge port of Ethernet3 exists")
}

func TestPort(t *testing.T) {
	s := read(t, leaf02())
	s.AdminUp = map[string]bool{"Ethernet0": true}

	requireDone(t, NewPort("Ethernet0", "9000", types.PortStatusUp).Applied(s))
	requirePending(t, NewPort("Ethernet0", "9000", types.PortStatusDown).Applied(s), "kernel Ethernet0 admin down")
	requirePending(t, NewPort("Ethernet9", "9000", types.PortStatusUp).Applied(s), "kernel Ethernet9 exists")
	requireDone(t, NewPort("Ethernet9", "9000", "").Applied(s))

	requirePending(t, NewPort("Ethernet0", "9000", "").Writable(s), "CONFIG_DB PORT|Ethernet0 exists")

	require.Equal(t, []string{"admin_status", "mtu"}, NewPort("Ethernet0", "9000", types.PortStatusUp).Owned())
	require.Equal(t, []string{"mtu"}, NewPort("Ethernet0", "9000", "").Owned())
}

func TestCurrent(t *testing.T) {
	keys := func(objects []Object) []string {
		var keys []string
		for _, o := range objects {
			keys = append(keys, db.NewDump("|", nil).String(o.Key()))
		}
		return keys
	}

	s := read(t, leaf02())
	require.ElementsMatch(t, []string{
		"VRF|Vrf20",
		"VLAN|Vlan1002",
		"SUPPRESS_VLAN_NEIGH|Vlan1002",
		"VLAN_INTERFACE|Vlan1002",
		"VXLAN_TUNNEL_MAP|vtep|map_20_Vlan1002",
		"INTERFACE|Ethernet3",
		"VLAN_MEMBER|Vlan4000|Ethernet0",
	}, keys(Current(s, []string{"Ethernet0", "Ethernet3"})), "the PXE vlan, its interface and other ports are not managed")

	// SONiC still has everything set up that was deleted from the CONFIG_DB
	data := leaf02()
	data["CONFIG_DB"] = test.StringMap{"VXLAN_TUNNEL": data["CONFIG_DB"]["VXLAN_TUNNEL"]}
	s = read(t, data)
	current := Current(s, []string{"Ethernet0", "Ethernet3"})
	require.ElementsMatch(t, []string{
		"VRF|Vrf20",
		"VLAN|Vlan1002",
		"SUPPRESS_VLAN_NEIGH|Vlan1002",
		"VLAN_INTERFACE|Vlan1002",
		"VXLAN_TUNNEL_MAP|vtep|map_20_Vlan1002",
		"INTERFACE|Ethernet3",
		"VLAN_MEMBER|Vlan4000|Ethernet0",
	}, keys(current))
	for _, o := range current {
		if ri, ok := o.(RouterInterface); ok {
			require.Equal(t, []db.Key{{"VRF", "Vrf20"}}, ri.Requires(), "the vrf is taken from the STATE_DB")
		}
	}

	vtep, err := VTEP(s)
	require.NoError(t, err)
	require.Equal(t, "vtep", vtep)
}
