// Package object describes the CONFIG_DB entries metal-core manages and how SONiC reports that it applied them.
//
// SONiC applies a CONFIG_DB entry asynchronously: a manager daemon (vrfmgrd, vlanmgrd, intfmgrd, vxlanmgrd) sets
// it up in the kernel and reports it in the STATE_DB, then orchagent programs it into the ASIC. Each daemon
// consumes its own tables, so the order in which entries are written is not the order in which they are applied.
// That is why every object declares what it requires and what tells that SONiC applied or removed it, and the
// order is derived from that.
package object

import (
	"fmt"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
)

type (
	// Object is one CONFIG_DB entry metal-core manages.
	Object interface {
		Key() db.Key
		// Fields are the fields the entry should have.
		Fields() db.Val
		// Owned are the fields metal-core manages, nil for all fields. Other fields of the entry are left alone.
		Owned() []string
		// Requires are the entries SONiC must have applied before this one is written.
		// They are only deleted once SONiC removed this one.
		Requires() []db.Key
		// Writable tells whether the entry may be written, besides what it requires.
		Writable(s *observe.Snapshot) Status
		// Replace tells whether SONiC cannot change the current fields of the entry into the desired ones, so the
		// entry has to be deleted and written again once SONiC removed it.
		Replace(current db.Val) bool
		// Applied tells whether SONiC applied the entry with its fields.
		Applied(s *observe.Snapshot) Status
		// Removed tells whether SONiC removed everything it set up for the entry. It is also checked after
		// the entry was deleted from the CONFIG_DB.
		Removed(s *observe.Snapshot) Status
		// Removable tells whether the entry may be deleted, besides that SONiC removed what requires it.
		Removable(s *observe.Snapshot, frr FRR) Status
	}

	// RoutesUser is an object whose removal depends on the routes of a vrf. The default vrf is "".
	RoutesUser interface {
		RoutesVrf() string
	}

	// FRR tells what the FRR configuration that was applied last uses.
	FRR interface {
		// UsesVrf tells whether FRR may still have the vrf configured.
		UsesVrf(vrf string) bool
		// UsesPort tells whether FRR may still peer on the port in the vrf. The default vrf is "".
		UsesPort(vrf, port string) bool
	}

	// Status lists what is pending, it is done once nothing is.
	Status struct {
		Pending []string
	}
)

func (s Status) Done() bool {
	return len(s.Pending) == 0
}

func (s Status) String() string {
	return strings.Join(s.Pending, ", ")
}

// Done is the status of something that is not waited for.
var Done = Status{}

// all is done once all statuses are.
func all(statuses ...Status) Status {
	var all Status
	for _, s := range statuses {
		all.Pending = append(all.Pending, s.Pending...)
	}
	return all
}

// check is pending with the description unless ok.
func check(ok bool, format string, args ...any) Status {
	if ok {
		return Done
	}
	return Status{Pending: []string{fmt.Sprintf(format, args...)}}
}

// exists is pending until the database has the entry.
func exists(database string, d db.Dump, key db.Key) Status {
	return check(d.Has(key), "%s %s exists", database, d.String(key))
}

// absent is pending until the database does not have the entry anymore.
func absent(database string, d db.Dump, key db.Key) Status {
	return check(!d.Has(key), "%s %s absent", database, d.String(key))
}

// entry implements the parts of Object that most entries share: all fields are owned, they can be changed in
// place, and the entry requires nothing and may be written and deleted any time.
type entry struct {
	key    db.Key
	fields db.Val
}

func (e entry) Key() db.Key                             { return e.key }
func (e entry) Fields() db.Val                          { return e.fields }
func (e entry) Owned() []string                         { return nil }
func (e entry) Requires() []db.Key                      { return nil }
func (e entry) Writable(*observe.Snapshot) Status       { return Done }
func (e entry) Replace(db.Val) bool                     { return false }
func (e entry) Removable(*observe.Snapshot, FRR) Status { return Done }
func (e entry) String() string                          { return strings.Join(e.key, "|") }
