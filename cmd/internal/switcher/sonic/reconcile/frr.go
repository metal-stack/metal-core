package reconcile

import (
	"slices"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/desired"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// Effective returns the config FRR is configured with. It differs from the config only for tenant vrfs:
//   - a vrf enters FRR once SONiC applied its entries, and a port of a machine once SONiC applied its router
//     interface in the vrf, so FRR configures what exists,
//   - both leave FRR as soon as they are not desired anymore, so FRR withdraws their routes before their entries
//     are deleted, see the Removable guards of the objects,
//   - what FRR has already applied stays, so a moment in which SONiC does not report it does not take it down.
//
// The ports to the spines, the firewalls and the management network are never waited for, a missing report of
// SONiC must not take down the BGP sessions metal-core reaches the metal-api with.
func Effective(cfg *types.Conf, sw *desired.Switch, s *observe.Snapshot, applied *types.Conf) *types.Conf {
	eff := *cfg
	eff.Ports.Vrfs = types.Vrfs{}

	for name, vrf := range cfg.Ports.Vrfs {
		previous := appliedVrf(applied, name)
		if previous == nil && !allApplied(sw, name, s) {
			continue
		}

		v := *vrf
		v.Neighbors = slices.DeleteFunc(slices.Clone(vrf.Neighbors), func(port string) bool {
			if previous != nil && slices.Contains(previous.Neighbors, port) {
				return false
			}
			return !sw.RouterInterfaces[port].Applied(s).Done()
		})
		eff.Ports.Vrfs[name] = &v
	}
	return &eff
}

func allApplied(sw *desired.Switch, vrf string, s *observe.Snapshot) bool {
	for _, o := range sw.Vrfs[vrf] {
		if !o.Applied(s).Done() {
			return false
		}
	}
	return true
}

func appliedVrf(applied *types.Conf, vrf string) *types.Vrf {
	if applied == nil {
		return nil
	}
	return applied.Ports.Vrfs[vrf]
}

// frrState tells what the FRR configuration that was applied last uses. Before FRR was applied, it may use anything.
type frrState struct {
	applied *types.Conf
}

func (f frrState) UsesVrf(vrf string) bool {
	return appliedVrf(f.applied, vrf) != nil || f.applied == nil
}

func (f frrState) UsesPort(vrf, port string) bool {
	if f.applied == nil {
		return true
	}
	if vrf != "" {
		v := appliedVrf(f.applied, vrf)
		return v != nil && slices.Contains(v.Neighbors, port)
	}
	if slices.Contains(f.applied.Ports.Underlay, port) {
		return true
	}
	for _, fw := range f.applied.Ports.Firewalls {
		if fw.Port == port {
			return true
		}
	}
	return false
}
