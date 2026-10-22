package types

import (
	"fmt"
	"net/netip"
)

type (
	Conf struct {
		Name                 string
		LogLevel             string
		Loopback             string
		ASN                  uint32
		Ports                Ports
		Vrfs                 Vrfs
		MetalCoreCIDR        string
		AdditionalMgmtRoutes []string
		Eth0                 MgmtInterface
		PXEVlanID            uint16
		SetSrcLoopback       bool
	}

	Ports map[string]*Port

	Port struct {
		Vrf         *string
		AdminStatus PortStatus
		MTU         uint16
		VLAN        uint16
	}

	Vrfs map[string]*Vrf

	Vrf struct {
		Filter
		VNI    uint32
		VLANID uint16
		Cidrs  []string
		Has4   bool
		Has6   bool
	}

	Firewall struct {
		Filter
		Port  string
		Cidrs []string
		Vnis  []string
	}

	Filter struct {
		IPPrefixLists []IPPrefixList
		RouteMaps     []RouteMap
	}

	MgmtInterface struct {
		AddressCIDR string
		Gateway     string
	}

	RouteMap struct {
		Name    string
		Entries []string
		Policy  Policy
		Order   int
	}

	IPPrefixList struct {
		AddressFamily AddressFamily
		Name          string
		Spec          string
	}

	cidrsByAf struct {
		ipv4Cidrs []string
		ipv6Cidrs []string
	}

	PortStatus    string
	AddressFamily string
	Policy        string
)

const (
	PortStatusUp   = PortStatus("up")
	PortStatusDown = PortStatus("down")

	AddressFamilyIPv4 = AddressFamily("ip")
	AddressFamilyIPv6 = AddressFamily("ipv6")

	PolicyPermit = Policy("permit")
	PolicyDeny   = Policy("deny")
)

func (s *Filter) Assemble(rmPrefix string, vnis, cidrs []string) {
	cidrsByAf := cidrsByAddressfamily(cidrs)
	if len(cidrsByAf.ipv4Cidrs) > 0 {
		prefixRouteMapName := fmt.Sprintf("%s-in", rmPrefix)
		prefixListName := fmt.Sprintf("%s-in-prefixes", rmPrefix)
		rm := RouteMap{
			Name:    prefixRouteMapName,
			Entries: []string{fmt.Sprintf("match ip address prefix-list %s", prefixListName)},
			Policy:  PolicyPermit,
			Order:   10,
		}
		s.RouteMaps = append(s.RouteMaps, rm)
		s.addPrefixList(prefixListName, cidrsByAf.ipv4Cidrs, AddressFamilyIPv4)
	}
	if len(cidrsByAf.ipv6Cidrs) > 0 {
		prefixRouteMapName := fmt.Sprintf("%s-in6", rmPrefix)
		prefixListName := fmt.Sprintf("%s-in6-prefixes", rmPrefix)
		rm := RouteMap{
			Name:    prefixRouteMapName,
			Entries: []string{fmt.Sprintf("match ipv6 address prefix-list %s", prefixListName)},
			Policy:  PolicyPermit,
			Order:   10,
		}
		s.RouteMaps = append(s.RouteMaps, rm)
		s.addPrefixList(prefixListName, cidrsByAf.ipv6Cidrs, AddressFamilyIPv6)
	}
	if len(vnis) > 0 {
		vniRouteMapName := fmt.Sprintf("%s-vni", rmPrefix)
		for j, vni := range vnis {
			rm := RouteMap{
				Name:    vniRouteMapName,
				Entries: []string{fmt.Sprintf("match evpn vni %s", vni)},
				Policy:  PolicyPermit,
				Order:   10 + j,
			}
			s.RouteMaps = append(s.RouteMaps, rm)
		}
	}
}

func (s *Filter) addPrefixList(prefixListName string, cidrs []string, af AddressFamily) {
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
