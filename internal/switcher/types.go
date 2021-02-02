package switcher

import (
	"fmt"

	"inet.af/netaddr"
)

// Conf holds the switch configuration
type Conf struct {
	Name                 string
	LogLevel             string
	Loopback             string
	ASN                  uint32
	Ports                Ports
	DevMode              bool
	MetalCoreCIDR        string
	AdditionalBridgeVIDs []string
	FrrTplFile           string
	InterfacesTplFile    string
}

type Ports struct {
	Eth0          Nic
	Underlay      []string
	Unprovisioned []string
	BladePorts    []string
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
	Name string
	Spec string
}

func (s *Filter) Assemble(rmPrefix string, vnis, cidrs []string) {
	cidrMap := sortCIDRByAddressfamily(cidrs)
	if len(cidrMap["ipv4"]) > 0 {
		prefixRouteMapName := fmt.Sprintf("%s-in", rmPrefix)
		prefixListName := fmt.Sprintf("%s-in-prefixes", rmPrefix)
		rm := RouteMap{
			Name:    prefixRouteMapName,
			Entries: []string{fmt.Sprintf("match ip address prefix-list %s", prefixListName)},
			Policy:  "permit",
			Order:   10,
		}
		s.RouteMaps = append(s.RouteMaps, rm)
		s.addPrefixList(prefixListName, cidrMap["ipv4"])
	}
	if len(cidrMap["ipv6"]) > 0 {
		prefixRouteMapName := fmt.Sprintf("%s-in6", rmPrefix)
		prefixListName := fmt.Sprintf("%s-in6-prefixes", rmPrefix)
		rm := RouteMap{
			Name:    prefixRouteMapName,
			Entries: []string{fmt.Sprintf("match ipv6 address prefix-list %s", prefixListName)},
			Policy:  "permit",
			Order:   10,
		}
		s.RouteMaps = append(s.RouteMaps, rm)
		s.addPrefixList(prefixListName, cidrMap["ipv6"])
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

func (s *Filter) addPrefixList(prefixListName string, cidrs []string) {
	for j, cidr := range cidrs {
		prefix, err := netaddr.ParseIPPrefix(cidr)
		if err != nil {
			continue
		}
		spec := fmt.Sprintf("seq %d permit %s le %d", 10+j, cidr, prefix.IP.BitLen())
		prefixList := IPPrefixList{
			Name: prefixListName,
			Spec: spec,
		}
		s.IPPrefixLists = append(s.IPPrefixLists, prefixList)
	}
}

func sortCIDRByAddressfamily(cidrs []string) map[string][]string {
	sortedCIDRs := make(map[string][]string)
	sortedCIDRs["ipv4"] = []string{}
	sortedCIDRs["ipv6"] = []string{}
	for _, cidr := range cidrs {
		prefix, err := netaddr.ParseIPPrefix(cidr)
		if err != nil {
			continue
		}
		if prefix.IP.Is4() {
			sortedCIDRs["ipv4"] = append(sortedCIDRs["ipv4"], cidr)
		}
		if prefix.IP.Is6() {
			sortedCIDRs["ipv6"] = append(sortedCIDRs["ipv6"], cidr)
		}
	}
	return sortedCIDRs
}
