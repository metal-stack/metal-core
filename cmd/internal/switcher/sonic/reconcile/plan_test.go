package reconcile

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/desired"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/object"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks/behaviors/vrfentries"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/sonictest"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// fakeFRR uses the vrfs and ports of the config it applied last.
type fakeFRR struct {
	applied *types.Conf
}

func (f *fakeFRR) UsesVrf(vrf string) bool {
	_, ok := f.applied.Ports.Vrfs[vrf]
	return ok
}

func (f *fakeFRR) UsesPort(vrf, port string) bool {
	if v, ok := f.applied.Ports.Vrfs[vrf]; ok {
		return slices.Contains(v.Neighbors, port)
	}
	return false
}

// harness runs the steps of a reconciliation against the fake SONiC daemons.
type harness struct {
	t          *testing.T
	databases  *sonictest.Databases
	daemons    *sonictest.Daemons
	db         *db.DB
	reader     *observe.Reader
	frr        *fakeFRR
	vrfEntries vrfentries.VrfEntries

	steps int
	// events are the deletes and writes, like "2 delete INTERFACE|Ethernet1"
	events []string
}

// leaf has the ports Ethernet0 to Ethernet5 and the PXE vlan.
func newHarness(t *testing.T, vrfEntries vrfentries.VrfEntries) *harness {
	ports := test.StringMap{}
	for i := range 6 {
		ports[fmt.Sprintf("Ethernet%d", i)] = test.StringMap{"admin_status": "up", "mtu": "9100"}
	}
	databases := sonictest.NewDatabases(t, sonictest.Data{
		"CONFIG_DB": {
			"PORT":         ports,
			"VLAN":         test.StringMap{"Vlan4000": test.StringMap{"vlanid": "4000"}},
			"VXLAN_TUNNEL": test.StringMap{"vtep": test.StringMap{"src_ip": "10.0.0.13"}},
		},
	})
	daemons := databases.Daemons()
	daemons.Sync(t)

	sonicDb := databases.DB(t)
	return &harness{
		t:          t,
		databases:  databases,
		daemons:    daemons,
		db:         sonicDb,
		reader:     observe.NewReader(sonicDb, daemons.Kernel),
		frr:        &fakeFRR{applied: &types.Conf{}},
		vrfEntries: vrfEntries,
	}
}

// step plans and executes one step, then lets the daemons apply it.
func (h *harness) step(cfg *types.Conf) Plan {
	h.t.Helper()
	ctx := h.t.Context()

	s, err := h.reader.Read(ctx, object.Tables)
	require.NoError(h.t, err)
	vtep, err := object.VTEP(s)
	require.NoError(h.t, err)
	sw, err := desired.Build(cfg, h.vrfEntries, vtep)
	require.NoError(h.t, err)
	current := object.Current(s, sw.Ports)
	require.NoError(h.t, h.reader.ReadRoutes(ctx, s, RouteVrfs(sw.Objects, current, s)))

	p := NewPlan(sw.Objects, current, s, h.frr)
	require.NoError(h.t, execute(ctx, slog.Default(), h.db.Config, p))

	h.steps++
	for _, k := range p.Deletes {
		h.events = append(h.events, fmt.Sprintf("%d delete %s", h.steps, keyString(k)))
	}
	for _, w := range p.Writes {
		h.events = append(h.events, fmt.Sprintf("%d write %s", h.steps, keyString(w.Key)))
	}

	h.daemons.Sync(h.t)
	return p
}

// converge runs steps until the plan is done, the config is what FRR applies right away.
func (h *harness) converge(cfg *types.Conf) {
	h.t.Helper()
	h.frr.applied = cfg
	for range 10 {
		if h.step(cfg).Done() {
			return
		}
	}
	require.Fail(h.t, "no convergence", "pending: %v", h.step(cfg).Pending)
}

// stepOf returns the step of the event, like "write VRF|Vrf46".
func (h *harness) stepOf(event string) int {
	h.t.Helper()
	for _, e := range h.events {
		step, rest, _ := strings.Cut(e, " ")
		if rest == event {
			var n int
			_, err := fmt.Sscan(step, &n)
			require.NoError(h.t, err)
			return n
		}
	}
	require.Fail(h.t, "event did not happen", "%s in %v", event, h.events)
	return 0
}

func (h *harness) requireBefore(first, second string) {
	h.t.Helper()
	require.Less(h.t, h.stepOf(first), h.stepOf(second), "%s before %s: %v", first, second, h.events)
}

func (h *harness) config() test.HashMap {
	return h.databases.Get(h.t, "CONFIG_DB")
}

func (h *harness) setRoute(key string, fields map[string]string) {
	h.t.Helper()
	vc, _ := h.databases.Client(h.t, "APPL_DB")
	cmd := vc.B().Hset().Key(key).FieldValue()
	for f, v := range fields {
		cmd = cmd.FieldValue(f, v)
	}
	require.NoError(h.t, vc.Do(h.t.Context(), cmd.Build()).Error())
}

func (h *harness) deleteRoute(key string) {
	h.t.Helper()
	vc, _ := h.databases.Client(h.t, "APPL_DB")
	require.NoError(h.t, vc.Do(h.t.Context(), vc.B().Del().Key(key).Build()).Error())
}

func pendingKeys(p Plan) []string {
	var keys []string
	for _, pending := range p.Pending {
		keys = append(keys, keyString(pending.Key))
	}
	return keys
}

// unprovisioned has all ports in the PXE vlan.
func unprovisioned() *types.Conf {
	return &types.Conf{
		PXEVlanID: 4000,
		Ports: types.Ports{
			Unprovisioned: []string{"Ethernet0", "Ethernet1", "Ethernet2", "Ethernet3", "Ethernet4", "Ethernet5"},
			Vrfs:          types.Vrfs{},
			Firewalls:     map[string]*types.Firewall{},
		},
	}
}

// machine has Ethernet1 in Vrf46 and the other ports in the PXE vlan.
func machine() *types.Conf {
	return &types.Conf{
		PXEVlanID: 4000,
		Ports: types.Ports{
			Unprovisioned: []string{"Ethernet0", "Ethernet2", "Ethernet3", "Ethernet4", "Ethernet5"},
			Vrfs: types.Vrfs{
				"Vrf46": {VNI: 46, VLANID: 1001, Neighbors: []string{"Ethernet1"}},
			},
			Firewalls: map[string]*types.Firewall{},
		},
	}
}

func TestPlan_CreateBottomUp(t *testing.T) {
	h := newHarness(t, vrfentries.Community{})
	h.converge(unprovisioned())
	h.events = nil

	h.converge(machine())

	h.requireBefore("delete VLAN_MEMBER|Vlan4000|Ethernet1", "write INTERFACE|Ethernet1")
	h.requireBefore("write VRF|Vrf46", "write INTERFACE|Ethernet1")
	h.requireBefore("write VRF|Vrf46", "write VLAN_INTERFACE|Vlan1001")
	h.requireBefore("write VLAN|Vlan1001", "write VLAN_INTERFACE|Vlan1001")
	h.requireBefore("write VLAN|Vlan1001", "write SUPPRESS_VLAN_NEIGH|Vlan1001")
	h.requireBefore("write VLAN|Vlan1001", "write VXLAN_TUNNEL_MAP|vtep|map_46_Vlan1001")
	h.requireBefore("write VRF|Vrf46", "write VXLAN_TUNNEL_MAP|vtep|map_46_Vlan1001")

	cfg := h.config()
	require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable", "vrf_name": "Vrf46"}, cfg["INTERFACE|Ethernet1"])
	require.Equal(t, map[string]string{"admin_status": "up", "mtu": "9000"}, cfg["PORT|Ethernet1"])
	require.NotContains(t, cfg, "VLAN_MEMBER|Vlan4000|Ethernet1")
}

// The teardown of the machine port waits for FRR and the routes, then removes the vrf from the top down.
func TestPlan_TeardownTopDown(t *testing.T) {
	for _, vrfEntries := range []vrfentries.VrfEntries{vrfentries.Community{}, vrfentries.L3Only{}} {
		t.Run(fmt.Sprintf("%T", vrfEntries), func(t *testing.T) {
			h := newHarness(t, vrfEntries)
			h.converge(machine())
			h.setRoute("ROUTE_TABLE:Vrf46:10.128.0.1", map[string]string{"ifname": "Ethernet1", "nexthop": "fe80::1"})
			h.setRoute("ROUTE_TABLE:Vrf46:fe80::/64", map[string]string{"ifname": "Ethernet1", "nexthop": "::"})
			h.events = nil

			// FRR still has the machine
			p := h.step(unprovisioned())
			for _, k := range p.Deletes {
				require.Equal(t, "SUPPRESS_VLAN_NEIGH", k[0], "only the ARP suppression, which no route uses, goes right away")
			}
			require.Contains(t, pendingKeys(p), "INTERFACE|Ethernet1")
			require.Contains(t, pendingKeys(p), "VRF|Vrf46")
			require.Contains(t, pendingKeys(p), "VLAN_MEMBER|Vlan4000|Ethernet1", "the port is still routed")

			// FRR removed the machine, but the route is not withdrawn yet
			h.frr.applied = unprovisioned()
			p = h.step(unprovisioned())
			require.NotContains(t, p.Deletes, db.Key{"INTERFACE", "Ethernet1"})
			require.Contains(t, fmt.Sprint(p.Pending), "ROUTE_TABLE:Vrf46:10.128.0.1")

			// the link-local route of the port stays until it left the vrf
			h.deleteRoute("ROUTE_TABLE:Vrf46:10.128.0.1")
			h.converge(unprovisioned())

			h.requireBefore("delete INTERFACE|Ethernet1", "write VLAN_MEMBER|Vlan4000|Ethernet1")
			h.requireBefore("delete INTERFACE|Ethernet1", "delete VRF|Vrf46")
			if _, ok := vrfEntries.(vrfentries.Community); ok {
				h.requireBefore("delete VLAN_INTERFACE|Vlan1001", "delete VRF|Vrf46")
				h.requireBefore("delete VXLAN_TUNNEL_MAP|vtep|map_46_Vlan1001", "delete VRF|Vrf46")
				h.requireBefore("delete VLAN_INTERFACE|Vlan1001", "delete VLAN|Vlan1001")
				h.requireBefore("delete SUPPRESS_VLAN_NEIGH|Vlan1001", "delete VLAN|Vlan1001")
				h.requireBefore("delete VXLAN_TUNNEL_MAP|vtep|map_46_Vlan1001", "delete VLAN|Vlan1001")
			}

			cfg := h.config()
			for key := range cfg {
				require.NotContains(t, key, "46", "everything of the vrf is gone")
				require.NotContains(t, key, "Vlan1001")
			}
			require.Contains(t, cfg, "VLAN_MEMBER|Vlan4000|Ethernet1")
			require.Equal(t, "up", cfg["PORT|Ethernet1"]["admin_status"], "the port was never taken down")
		})
	}
}

// A router interface SONiC does not remove keeps its port and vrf, but not the other ports.
func TestPlan_StuckPortDoesNotBlockOthers(t *testing.T) {
	h := newHarness(t, vrfentries.L3Only{})
	h.converge(machine())
	h.daemons.Stuck["INTERFACE|Ethernet1"] = true

	freed := unprovisioned()
	freed.Ports.Unprovisioned = []string{"Ethernet0", "Ethernet1", "Ethernet3", "Ethernet4", "Ethernet5"}
	freed.Ports.Underlay = []string{"Ethernet2"}
	h.frr.applied = freed
	for range 5 {
		h.step(freed)
	}

	cfg := h.config()
	require.NotContains(t, cfg, "INTERFACE|Ethernet1", "the entry is deleted")
	require.NotContains(t, cfg, "VLAN_MEMBER|Vlan4000|Ethernet1", "the port is still routed")
	require.Contains(t, cfg, "VRF|Vrf46", "the vrf is still required by the router interface")
	require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable"}, cfg["INTERFACE|Ethernet2"], "other ports are configured")

	p := h.step(freed)
	require.Contains(t, fmt.Sprint(p.Pending), "INTERFACE|Ethernet1 is deleted, waits until SONiC removed it")
	require.Contains(t, fmt.Sprint(p.Pending), "VRF|Vrf46 is required by INTERFACE|Ethernet1")

	delete(h.daemons.Stuck, "INTERFACE|Ethernet1")
	h.converge(freed)
	require.NotContains(t, h.config(), "VRF|Vrf46")
	require.Contains(t, h.config(), "VLAN_MEMBER|Vlan4000|Ethernet1")
}

// A port that moves into another vrf leaves the previous one first.
func TestPlan_ReplaceVrfOfPort(t *testing.T) {
	h := newHarness(t, vrfentries.L3Only{})
	h.converge(machine())
	h.events = nil

	moved := machine()
	moved.Ports.Vrfs = types.Vrfs{"Vrf47": {VNI: 47, VLANID: 1002, Neighbors: []string{"Ethernet1"}}}
	h.converge(moved)

	h.requireBefore("delete INTERFACE|Ethernet1", "write INTERFACE|Ethernet1")
	h.requireBefore("write VRF|Vrf47", "write INTERFACE|Ethernet1")
	h.requireBefore("delete INTERFACE|Ethernet1", "delete VRF|Vrf46")
	require.Equal(t, "Vrf47", h.config()["INTERFACE|Ethernet1"]["vrf_name"])
	require.NotContains(t, h.config(), "VRF|Vrf46")
}

// A firewall port is a router interface in the default vrf and becomes one in the vrf of a machine.
func TestPlan_FirewallBecomesMachine(t *testing.T) {
	h := newHarness(t, vrfentries.Community{})
	fw := unprovisioned()
	fw.Ports.Unprovisioned = []string{"Ethernet0", "Ethernet2", "Ethernet3", "Ethernet4", "Ethernet5"}
	fw.Ports.Firewalls = map[string]*types.Firewall{"Ethernet1": {Port: "Ethernet1"}}
	h.converge(fw)
	require.Equal(t, map[string]string{"admin_status": "up", "mtu": "9216"}, h.config()["PORT|Ethernet1"])
	h.events = nil

	h.converge(machine())
	h.requireBefore("delete INTERFACE|Ethernet1", "write INTERFACE|Ethernet1")
	require.Equal(t, "Vrf46", h.config()["INTERFACE|Ethernet1"]["vrf_name"])
}

// A vrf that is allocated again right after it was freed is written again once SONiC removed it.
func TestPlan_VrfAgainRightAfterRemoval(t *testing.T) {
	h := newHarness(t, vrfentries.L3Only{})
	h.converge(machine())
	h.frr.applied = unprovisioned()
	h.events = nil

	h.daemons.Stuck["VRF|Vrf46"] = true
	for range 3 {
		h.step(unprovisioned())
	}
	require.NotContains(t, h.config(), "VRF|Vrf46")

	p := h.step(machine())
	require.Contains(t, fmt.Sprint(p.Pending), "VRF|Vrf46 waits until SONiC removed the previous entry")
	require.Contains(t, fmt.Sprint(p.Pending), "INTERFACE|Ethernet1 waits for VRF|Vrf46")

	delete(h.daemons.Stuck, "VRF|Vrf46")
	h.converge(machine())
	h.requireBefore("delete VRF|Vrf46", "write VRF|Vrf46")
}

func TestPlan_InSyncIsDone(t *testing.T) {
	h := newHarness(t, vrfentries.Community{})
	h.converge(machine())

	p := h.step(machine())
	require.True(t, p.Done(), "%+v", p)
}

// The admin status of a port that is not applied by the kernel is reported.
func TestPlan_AdminStatusNotApplied(t *testing.T) {
	h := newHarness(t, vrfentries.Community{})
	h.converge(unprovisioned())

	down := unprovisioned()
	down.Ports.AdminStatus = map[string]types.PortStatus{"Ethernet3": types.PortStatusDown}
	h.daemons.Stuck["PORT|Ethernet3"] = true
	h.step(down)
	p := h.step(down)

	require.Equal(t, []string{"PORT|Ethernet3"}, pendingKeys(p))
	require.Contains(t, p.Pending[0].Reason, "kernel Ethernet3 admin down")
}
