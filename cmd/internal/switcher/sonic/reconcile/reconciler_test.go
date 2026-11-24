package reconcile

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/desired"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/object"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/quirks"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// bgpFRR peers with the machines it is configured with. A machine announces its route once FRR peers with it on
// a router interface SONiC applied, and the route is withdrawn once FRR no longer peers with it.
type bgpFRR struct {
	h *harness
	// applies are what the CONFIG_DB and STATE_DB looked like each time FRR was applied
	applies []frrApply
	err     error
}

type frrApply struct {
	cfg    *types.Conf
	config map[string]map[string]string
	state  map[string]map[string]string
}

// machineRoute is the route each machine announces, by port.
func machineRoute(vrf, port string) string {
	return "ROUTE_TABLE:" + vrf + ":10.128.0." + strings.TrimPrefix(port, "Ethernet")
}

func (f *bgpFRR) Apply(_ context.Context, cfg *types.Conf) error {
	if f.err != nil {
		return f.err
	}
	f.applies = append(f.applies, frrApply{cfg: cfg, config: f.h.config(), state: f.h.databases.Get(f.h.t, "STATE_DB")})

	appl := f.h.databases.Get(f.h.t, "APPL_DB")
	for key := range appl {
		if !strings.HasPrefix(key, "ROUTE_TABLE:Vrf") || strings.Contains(key, ":fe80:") {
			continue
		}
		vrf, rest, _ := strings.Cut(strings.TrimPrefix(key, "ROUTE_TABLE:"), ":")
		port := "Ethernet" + rest[strings.LastIndex(rest, ".")+1:]
		if v, ok := cfg.Ports.Vrfs[vrf]; !ok || !slices.Contains(v.Neighbors, port) {
			f.h.deleteRoute(key)
		}
	}
	for vrf, v := range cfg.Ports.Vrfs {
		for _, port := range v.Neighbors {
			f.h.setRoute(machineRoute(vrf, port), map[string]string{"ifname": port, "nexthop": "fe80::1"})
		}
	}
	return nil
}

func newReconciler(t *testing.T, h *harness) (*Reconciler, *bgpFRR) {
	frr := &bgpFRR{h: h}
	r := New(slog.Default(), h.db, h.daemons.Kernel, quirks.QuirkProfiles["asternos"](), frr, WithPause(func(context.Context, time.Duration) error {
		h.daemons.Sync(t)
		return nil
	}))
	r.settle = 50 * time.Millisecond
	return r, frr
}

func TestReconciler_FRROnTopOfSONiC(t *testing.T) {
	h := newHarness(t, quirks.QuirkProfiles["asternos"]().VrfEntries)
	r, frr := newReconciler(t, h)

	require.NoError(t, r.Apply(t.Context(), unprovisioned()))
	require.NoError(t, r.Apply(t.Context(), machine()))

	// FRR peers with the machine once SONiC applied the router interface in the vrf
	var peering *frrApply
	for i := range frr.applies {
		if v, ok := frr.applies[i].cfg.Ports.Vrfs["Vrf46"]; ok && slices.Contains(v.Neighbors, "Ethernet1") {
			peering = &frr.applies[i]
			break
		}
	}
	require.NotNil(t, peering)
	require.Equal(t, "Vrf46", peering.state["INTERFACE_TABLE|Ethernet1"]["vrf"])
	require.Contains(t, h.databases.Get(t, "APPL_DB"), machineRoute("Vrf46", "Ethernet1"))

	// FRR stops peering before the router interface is deleted
	frr.applies = nil
	require.NoError(t, r.Apply(t.Context(), unprovisioned()))
	require.NotEmpty(t, frr.applies)
	first := frr.applies[0]
	require.NotContains(t, first.cfg.Ports.Vrfs, "Vrf46")
	require.Contains(t, first.config, "INTERFACE|Ethernet1")
	require.Contains(t, first.config, "VRF|Vrf46")

	require.NotContains(t, h.config(), "INTERFACE|Ethernet1")
	require.NotContains(t, h.config(), "VRF|Vrf46")
	require.Contains(t, h.config(), "VLAN_MEMBER|Vlan4000|Ethernet1")
}

func TestReconciler_InSync(t *testing.T) {
	h := newHarness(t, quirks.QuirkProfiles["asternos"]().VrfEntries)
	r, frr := newReconciler(t, h)
	require.NoError(t, r.Apply(t.Context(), machine()))

	frr.applies = nil
	require.NoError(t, r.Apply(t.Context(), machine()))
	require.Empty(t, frr.applies, "FRR is not reloaded without a change")
}

func TestReconciler_StuckIsPending(t *testing.T) {
	h := newHarness(t, quirks.QuirkProfiles["asternos"]().VrfEntries)
	r, _ := newReconciler(t, h)
	require.NoError(t, r.Apply(t.Context(), machine()))

	h.daemons.Stuck["INTERFACE|Ethernet1"] = true
	freed := unprovisioned()
	freed.Ports.Unprovisioned = []string{"Ethernet0", "Ethernet1", "Ethernet3", "Ethernet4", "Ethernet5"}
	freed.Ports.Underlay = []string{"Ethernet2"}

	err := r.Apply(t.Context(), freed)
	var pending *PendingError
	require.ErrorAs(t, err, &pending)
	require.Contains(t, err.Error(), "INTERFACE|Ethernet1 is deleted, waits until SONiC removed it")
	require.Contains(t, err.Error(), "VRF|Vrf46 is required by INTERFACE|Ethernet1")
	require.Equal(t, map[string]string{"ipv6_use_link_local_only": "enable"}, h.config()["INTERFACE|Ethernet2"], "other ports are configured")

	delete(h.daemons.Stuck, "INTERFACE|Ethernet1")
	require.NoError(t, r.Apply(t.Context(), freed))
	require.NotContains(t, h.config(), "VRF|Vrf46")
}

func TestReconciler_FRRFails(t *testing.T) {
	h := newHarness(t, quirks.QuirkProfiles["asternos"]().VrfEntries)
	r, frr := newReconciler(t, h)
	require.NoError(t, r.Apply(t.Context(), machine()))

	frr.err = errors.New("frr-reload failed")
	require.ErrorContains(t, r.Apply(t.Context(), unprovisioned()), "frr-reload failed")
	require.Contains(t, h.config(), "INTERFACE|Ethernet1", "FRR may still peer on the port")

	frr.err = nil
	require.NoError(t, r.Apply(t.Context(), unprovisioned()))
	require.NotContains(t, h.config(), "INTERFACE|Ethernet1")
}

// A vrf and its machine enter FRR once SONiC applied them, and stay once FRR has them.
func TestEffective(t *testing.T) {
	h := newHarness(t, quirks.QuirkProfiles["asternos"]().VrfEntries)
	s, err := h.reader.Read(t.Context(), object.Tables)
	require.NoError(t, err)
	sw, err := desired.Build(machine(), h.vrfEntries, "vtep")
	require.NoError(t, err)

	eff := Effective(machine(), sw, s, nil)
	require.Empty(t, eff.Ports.Vrfs, "SONiC has not applied the vrf")
	require.Equal(t, machine().Ports.Unprovisioned, eff.Ports.Unprovisioned, "other ports are not waited for")

	eff = Effective(machine(), sw, s, machine())
	require.Equal(t, []string{"Ethernet1"}, eff.Ports.Vrfs["Vrf46"].Neighbors, "a moment in which SONiC does not report the vrf does not take it out of FRR")

	require.Empty(t, Effective(unprovisioned(), sw, s, machine()).Ports.Vrfs, "a vrf that is not desired leaves FRR right away")
}
