package object

import (
	"slices"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/observe"
)

// programmed tells whether orchagent programs the route into the ASIC, where it references the router interfaces
// of its next hops and the virtual router of its vrf.
//
// orchagent does not program IPv6 link-local routes. So the connected fe80::/64 that fpmsyncd installs for every
// port of a vrf references nothing, and only goes away once the port left the vrf. Waiting for it would never end.
// Observed on Edgecore SONiC and AsterNOS.
func programmed(r db.Route) bool {
	return !r.Prefix.Addr().IsLinkLocalUnicast()
}

// via tells whether the port is one of the next hops of the route.
func via(r db.Route, port string) bool {
	return slices.ContainsFunc(r.NextHops, func(nh db.NextHop) bool {
		return nh.Interface == port
	})
}

// noRoutesIn is pending while orchagent has routes in the vrf, they reference its virtual router.
func noRoutesIn(s *observe.Snapshot, vrf string) Status {
	return noRoutes(s, vrf, func(db.Route) bool { return true }, "in "+vrfString(vrf))
}

// noRoutesVia is pending while orchagent has routes via the port in the vrf, they reference its router interface.
func noRoutesVia(s *observe.Snapshot, vrf, port string) Status {
	return noRoutes(s, vrf, func(r db.Route) bool { return via(r, port) }, "via "+port+" in "+vrfString(vrf))
}

func noRoutes(s *observe.Snapshot, vrf string, match func(db.Route) bool, desc string) Status {
	routes, read := s.Routes(vrf)
	if !read {
		return check(false, "routes of %s read", vrfString(vrf))
	}

	var keys []string
	for _, r := range routes {
		if programmed(r) && match(r) {
			keys = append(keys, r.Key)
		}
	}
	return check(len(keys) == 0, "APPL_DB no routes %s (%s)", desc, strings.Join(keys, ", "))
}

func vrfString(vrf string) string {
	if vrf == "" {
		return "default vrf"
	}
	return vrf
}
