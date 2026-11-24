// Package observe reads the state of a SONiC switch as of one point in time.
//
// A snapshot holds the configuration metal-core wrote and what the SONiC daemons report about applying it,
// so deciding what to do next needs no further reads.
package observe

import (
	"context"
	"fmt"
	"net"
	"slices"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

type (
	// Snapshot is the state of the switch as of one point in time.
	Snapshot struct {
		Config db.Dump
		State  db.Dump
		Appl   db.Dump
		Asic   db.Dump
		// PortOIDs and RifOIDs map port names to the oids of their SAI port and router interface, from the COUNTERS_DB.
		PortOIDs map[string]db.OID
		RifOIDs  map[string]db.OID
		// AdminUp is the admin status of the netdevs as the kernel reports it, by name.
		AdminUp map[string]bool

		// routes are the routes in the APPL_DB by vrf, the default vrf is "". Only read vrfs are present.
		routes map[string][]db.Route
	}

	// Tables are the tables a snapshot holds, by database.
	Tables struct {
		Config []db.Key
		State  []db.Key
		Appl   []db.Key
		Asic   []db.Key
	}

	// Kernel reports the admin status of the netdevs, by name.
	Kernel interface {
		AdminUp() (map[string]bool, error)
	}

	Reader struct {
		db     *db.DB
		kernel Kernel
	}
)

func NewReader(sonicDb *db.DB, kernel Kernel) *Reader {
	return &Reader{db: sonicDb, kernel: kernel}
}

// Read reads the tables and the admin status of the netdevs.
func (r *Reader) Read(ctx context.Context, tables Tables) (*Snapshot, error) {
	var (
		s   = &Snapshot{routes: map[string][]db.Route{}}
		err error
	)

	if s.Config, err = r.db.Config.Dump(ctx, tables.Config...); err != nil {
		return nil, fmt.Errorf("could not read CONFIG_DB: %w", err)
	}
	if s.State, err = r.db.State.Dump(ctx, tables.State...); err != nil {
		return nil, fmt.Errorf("could not read STATE_DB: %w", err)
	}
	if s.Appl, err = r.db.Appl.Dump(ctx, tables.Appl...); err != nil {
		return nil, fmt.Errorf("could not read APPL_DB: %w", err)
	}
	if s.Asic, err = r.db.Asic.Dump(ctx, tables.Asic...); err != nil {
		return nil, fmt.Errorf("could not read ASIC_DB: %w", err)
	}
	if s.PortOIDs, err = r.db.Counters.GetPortNameMap(ctx); err != nil {
		return nil, fmt.Errorf("could not read port oids: %w", err)
	}
	if s.RifOIDs, err = r.db.Counters.GetRifNameMap(ctx); err != nil {
		return nil, fmt.Errorf("could not read router interface oids: %w", err)
	}

	if s.AdminUp, err = r.kernel.AdminUp(); err != nil {
		return nil, fmt.Errorf("could not read the admin status of the netdevs: %w", err)
	}

	return s, nil
}

// ReadRoutes adds the routes of the vrfs to the snapshot. The default vrf is "".
func (r *Reader) ReadRoutes(ctx context.Context, s *Snapshot, vrfs []string) error {
	for _, vrf := range vrfs {
		if _, read := s.routes[vrf]; read {
			continue
		}

		// the routes of the default vrf can only be read together with the ones of all vrfs
		routes, err := r.db.Appl.GetRoutes(ctx, vrf)
		if err != nil {
			return fmt.Errorf("could not read routes of vrf %q: %w", vrf, err)
		}
		if vrf == "" {
			routes = slices.DeleteFunc(routes, func(r db.Route) bool { return r.Vrf != "" })
		}
		s.SetRoutes(vrf, routes)
	}
	return nil
}

// Routes returns the routes of the vrf and whether they were read.
func (s *Snapshot) Routes(vrf string) ([]db.Route, bool) {
	routes, read := s.routes[vrf]
	return routes, read
}

// SetRoutes sets the routes of the vrf.
func (s *Snapshot) SetRoutes(vrf string, routes []db.Route) {
	if s.routes == nil {
		s.routes = map[string][]db.Route{}
	}
	s.routes[vrf] = routes
}

// NetKernel reads the admin status from the netdevs of this host.
type NetKernel struct{}

func (NetKernel) AdminUp() (map[string]bool, error) {
	nics, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	up := make(map[string]bool, len(nics))
	for _, nic := range nics {
		up[nic.Name] = nic.Flags&net.FlagUp != 0
	}
	return up, nil
}
