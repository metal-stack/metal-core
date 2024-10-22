package types

import (
	"fmt"
	"net/netip"
)

// Conf holds the switch configuration
type Conf struct {
	Name                    string
	LogLevel                string
	Loopback                string
	ASN                     uint32
	Ports                   Ports
	MetalCoreCIDR           string
	AdditionalBridgeVIDs    []string
	PXEVlanID               uint16
	AdditionalRouteMapCIDRs []string
}

type Ports struct {
	Eth0          Nic
	Underlay      []string
	Unprovisioned []string
	BladePorts    []string
	DownPorts     map[string]bool
	Vrfs          map[string]*Vrf
	Firewalls     map[string]*Firewall
}

// Tenant holds the switch configuration for a specific tenant
type Vrf struct {
	Filter
	VNI       uint32
	VLANID    uint16
	Neighbors []string
	Cidrs     []string
	Has4      bool
	Has6      bool
}

type Firewall struct {
	Filter
	Port  string
	Cidrs []string
	Vnis  []string
}

type Filter struct {
	IPPrefixLists []IPPrefixList
	RouteMaps     []RouteMap
}

// Nic holds the configuration for a network interface
type Nic struct {
	AddressCIDR string
	Gateway     string
}

// RouteMap represents a route-map to permit or deny routes.
type RouteMap struct {
	Name    string
	Entries []string
	Policy  string
	Order   int
}

// IPPrefixList represents 'ip prefix-list' filtering mechanism to be used in combination with route-maps.
type IPPrefixList struct {
	AddressFamily string
	Name          string
	Spec          string
}

func (s *Filter) Assemble(rmPrefix string, vnis, cidrs []string) {
	cidrsByAf := cidrsByAddressfamily(cidrs)
	if len(cidrsByAf.ipv4Cidrs) > 0 {
		prefixRouteMapName := fmt.Sprintf("%s-in", rmPrefix)
		prefixListName := fmt.Sprintf("%s-in-prefixes", rmPrefix)
		rm := RouteMap{
			Name:    prefixRouteMapName,
			Entries: []string{fmt.Sprintf("match ip address prefix-list %s", prefixListName)},
			Policy:  "permit",
			Order:   10,
		}
		s.RouteMaps = append(s.RouteMaps, rm)
		s.addPrefixList(prefixListName, cidrsByAf.ipv4Cidrs, "ip")
	}
	if len(cidrsByAf.ipv6Cidrs) > 0 {
		prefixRouteMapName := fmt.Sprintf("%s-in6", rmPrefix)
		prefixListName := fmt.Sprintf("%s-in6-prefixes", rmPrefix)
		rm := RouteMap{
			Name:    prefixRouteMapName,
			Entries: []string{fmt.Sprintf("match ipv6 address prefix-list %s", prefixListName)},
			Policy:  "permit",
			Order:   10,
		}
		s.RouteMaps = append(s.RouteMaps, rm)
		s.addPrefixList(prefixListName, cidrsByAf.ipv6Cidrs, "ipv6")
	}
	if len(vnis) > 0 {
		vniRouteMapName := fmt.Sprintf("%s-vni", rmPrefix)
		for j, vni := range vnis {
			rm := RouteMap{
				Name:    vniRouteMapName,
				Entries: []string{fmt.Sprintf("match evpn vni %s", vni)},
				Policy:  "permit",
				Order:   10 + j,
			}
			s.RouteMaps = append(s.RouteMaps, rm)
		}
	}
}

func (s *Filter) addPrefixList(prefixListName string, cidrs []string, af string) {
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		spec := fmt.Sprintf("permit %s le %d", cidr, prefix.Addr().BitLen())
		prefixList := IPPrefixList{
			AddressFamily: af,
			Name:          prefixListName,
			Spec:          spec,
		}
		s.IPPrefixLists = append(s.IPPrefixLists, prefixList)
	}
}

type cidrsByAf struct {
	ipv4Cidrs []string
	ipv6Cidrs []string
}

func cidrsByAddressfamily(cidrs []string) cidrsByAf {
	cs := cidrsByAf{
		ipv4Cidrs: []string{},
		ipv6Cidrs: []string{},
	}
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		if prefix.Addr().Is4() {
			cs.ipv4Cidrs = append(cs.ipv4Cidrs, cidr)
		}
		if prefix.Addr().Is6() {
			cs.ipv6Cidrs = append(cs.ipv6Cidrs, cidr)
		}
	}
	return cs
}
