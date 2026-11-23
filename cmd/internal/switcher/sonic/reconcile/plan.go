// Package reconcile brings the CONFIG_DB and FRR of a SONiC switch to the desired state, one step at a time.
//
// Each step plans on a snapshot of the switch, so every decision is made on what SONiC reports, never on what
// metal-core wrote before. A step writes what can be written and deletes what can be deleted. What has to wait
// for SONiC is pending, the next step continues with it.
package reconcile

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/object"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
)

type (
	// Plan is what one step does, and what waits for SONiC.
	Plan struct {
		Writes  []Write
		Deletes []db.Key
		Pending []Pending
	}

	// Write sets the fields of an entry and deletes the managed fields it should not have.
	Write struct {
		Key    db.Key
		Set    db.Val
		Delete []string
	}

	// Pending is an entry that waits for SONiC.
	Pending struct {
		Key    db.Key
		Reason string
	}
)

func (p Pending) String() string {
	return fmt.Sprintf("%s %s", strings.Join(p.Key, "|"), p.Reason)
}

// Done tells whether nothing is left to do or to wait for.
func (p Plan) Done() bool {
	return len(p.Writes) == 0 && len(p.Deletes) == 0 && len(p.Pending) == 0
}

// NewPlan plans the next step from the desired objects to the current ones, see object.Current.
//
// A desired entry is written once SONiC applied the entries it requires and nothing keeps it from being written.
// An entry that is not desired is deleted once SONiC removed the entries that require it and nothing keeps it
// from being deleted. An entry SONiC cannot change in place is deleted, and written again once SONiC removed it.
func NewPlan(desired, current []object.Object, s *observe.Snapshot, frr object.FRR) Plan {
	var (
		p        Plan
		wanted   = byKey(desired)
		existing = byKey(current)
	)

	for _, c := range deletions(desired, current, s) {
		if dependents := requiredBy(c, current); len(dependents) > 0 {
			p.pending(c, "is required by %s", strings.Join(dependents, ", "))
			continue
		}
		if st := c.Removable(s, frr); !st.Done() {
			p.pending(c, "waits for %s", st)
			continue
		}
		p.Deletes = append(p.Deletes, c.Key())
	}

	// a deleted entry stays current until SONiC removed it
	for _, c := range current {
		if _, ok := wanted[keyString(c.Key())]; ok || s.Config.Has(c.Key()) {
			continue
		}
		if st := c.Removed(s); !st.Done() {
			p.pending(c, "is deleted, waits until SONiC removed it: %s", st)
		}
	}

	ready := map[string]bool{}
	for _, d := range desired {
		current, configured := s.Config.Get(d.Key())
		set, del := diff(d, current)
		inSync := configured && len(set) == 0 && len(del) == 0
		if inSync {
			if st := d.Applied(s); !st.Done() {
				p.pending(d, "waits until SONiC applied it: %s", st)
				continue
			}
			ready[keyString(d.Key())] = true
		}
	}

	for _, d := range desired {
		key := keyString(d.Key())
		current, configured := s.Config.Get(d.Key())
		set, del := diff(d, current)

		switch {
		case configured && len(set) == 0 && len(del) == 0:
			// in sync, applied or pending above
			continue
		case configured && d.Replace(current):
			p.pending(d, "is replaced, waits until it is deleted")
			continue
		case !configured && existing[key] != nil:
			if st := existing[key].Removed(s); !st.Done() {
				p.pending(d, "waits until SONiC removed the previous entry: %s", st)
				continue
			}
		}

		if missing := unready(d.Requires(), wanted, ready); len(missing) > 0 {
			p.pending(d, "waits for %s", strings.Join(missing, ", "))
			continue
		}
		if st := d.Writable(s); !st.Done() {
			p.pending(d, "waits for %s", st)
			continue
		}
		p.Writes = append(p.Writes, Write{Key: d.Key(), Set: set, Delete: del})
	}

	return p
}

// RouteVrfs returns the vrfs whose routes are needed to plan the next step, as they decide whether entries can be deleted.
func RouteVrfs(desired, current []object.Object, s *observe.Snapshot) []string {
	var vrfs []string
	for _, c := range deletions(desired, current, s) {
		if user, ok := c.(object.RoutesUser); ok && !slices.Contains(vrfs, user.RoutesVrf()) {
			vrfs = append(vrfs, user.RoutesVrf())
		}
	}
	return vrfs
}

// deletions returns the entries in the CONFIG_DB that are not desired, or that have to be replaced.
func deletions(desired, current []object.Object, s *observe.Snapshot) []object.Object {
	wanted := byKey(desired)

	var deletions []object.Object
	for _, c := range current {
		fields, configured := s.Config.Get(c.Key())
		if !configured {
			continue
		}
		if d, ok := wanted[keyString(c.Key())]; ok && !d.Replace(fields) {
			continue
		}
		deletions = append(deletions, c)
	}
	return deletions
}

// requiredBy returns the current entries that require the entry, also the ones SONiC has not removed yet.
func requiredBy(o object.Object, current []object.Object) []string {
	var dependents []string
	for _, c := range current {
		if slices.ContainsFunc(c.Requires(), func(k db.Key) bool { return slices.Equal(k, o.Key()) }) {
			dependents = append(dependents, keyString(c.Key()))
		}
	}
	return dependents
}

// unready returns the required entries that are desired but not applied. Entries metal-core does not manage,
// like the PXE vlan, are part of the switch configuration.
func unready(requires []db.Key, wanted map[string]object.Object, ready map[string]bool) []string {
	var missing []string
	for _, k := range requires {
		key := keyString(k)
		if _, managed := wanted[key]; managed && !ready[key] {
			missing = append(missing, key)
		}
	}
	return missing
}

// diff returns the fields to set and the managed fields to delete to get from the current fields to the desired ones.
func diff(o object.Object, current db.Val) (db.Val, []string) {
	set := db.Val{}
	for field, value := range o.Fields() {
		if v, ok := current[field]; !ok || v != value {
			set[field] = value
		}
	}

	owned := o.Owned()
	if owned == nil {
		owned = slices.Collect(maps.Keys(current))
	}
	var del []string
	for _, field := range owned {
		_, desired := o.Fields()[field]
		_, has := current[field]
		if has && !desired {
			del = append(del, field)
		}
	}
	slices.Sort(del)
	return set, del
}

func (p *Plan) pending(o object.Object, format string, args ...any) {
	p.Pending = append(p.Pending, Pending{Key: o.Key(), Reason: fmt.Sprintf(format, args...)})
}

func byKey(objects []object.Object) map[string]object.Object {
	m := make(map[string]object.Object, len(objects))
	for _, o := range objects {
		m[keyString(o.Key())] = o
	}
	return m
}

func keyString(k db.Key) string {
	return strings.Join(k, "|")
}
