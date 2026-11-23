package db

import (
	"fmt"
	"net/netip"
	"strings"
)

type (
	// Route is a route fpmsyncd installed in the APPL_DB ROUTE_TABLE.
	Route struct {
		Key string
		// Vrf is empty for the default vrf
		Vrf      string
		Prefix   netip.Prefix
		NextHops []NextHop
	}

	// NextHop is one next hop of a route. The gateway is "0.0.0.0" or "::" for a directly connected route.
	NextHop struct {
		Gateway   string
		Interface string
	}
)

// parseRoute parses a ROUTE_TABLE entry. Next hops are either part of the entry (ifname and nexthop) or,
// with next hop groups, part of the referenced NEXTHOP_GROUP_TABLE entry, which is passed as group.
func parseRoute(key string, val Val, group Val) (Route, error) {
	vrf, prefix, err := parseRouteKey(key)
	if err != nil {
		return Route{}, err
	}

	hops := val
	if val["nexthop_group"] != "" {
		hops = group
	}

	return Route{
		Key:      key,
		Vrf:      vrf,
		Prefix:   prefix,
		NextHops: parseNextHops(hops["ifname"], hops["nexthop"]),
	}, nil
}

// parseRouteKey splits a key like ROUTE_TABLE:Vrf20:10.0.0.1 or ROUTE_TABLE:fe80::/64 into vrf and prefix.
// SONiC requires vrf names to start with "Vrf", which tells them apart from the colons of an IPv6 prefix.
func parseRouteKey(key string) (vrf string, prefix netip.Prefix, err error) {
	rest, ok := strings.CutPrefix(key, "ROUTE_TABLE:")
	if !ok {
		return "", netip.Prefix{}, fmt.Errorf("%s is no route", key)
	}
	if strings.HasPrefix(rest, "Vrf") {
		vrf, rest, _ = strings.Cut(rest, ":")
	}

	// host routes have no prefix length
	if !strings.Contains(rest, "/") {
		addr, err := netip.ParseAddr(rest)
		if err != nil {
			return "", netip.Prefix{}, fmt.Errorf("could not parse prefix of route %s: %w", key, err)
		}
		return vrf, netip.PrefixFrom(addr, addr.BitLen()), nil
	}

	prefix, err = netip.ParsePrefix(rest)
	if err != nil {
		return "", netip.Prefix{}, fmt.Errorf("could not parse prefix of route %s: %w", key, err)
	}
	return vrf, prefix, nil
}

// parseNextHops pairs the comma separated interfaces and gateways of an ECMP route.
func parseNextHops(ifnames, gateways string) []NextHop {
	if ifnames == "" {
		return nil
	}

	interfaces := strings.Split(ifnames, ",")
	var gws []string
	if gateways != "" {
		gws = strings.Split(gateways, ",")
	}

	hops := make([]NextHop, 0, len(interfaces))
	for i, ifname := range interfaces {
		hop := NextHop{Interface: ifname}
		if i < len(gws) {
			hop.Gateway = gws[i]
		}
		hops = append(hops, hop)
	}
	return hops
}
