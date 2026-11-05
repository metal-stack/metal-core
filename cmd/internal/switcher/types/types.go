package types

import (
	"net/netip"
)

type (
	Conf struct {
		Name                 string
		LogLevel             string
		Loopback             string
		ASN                  uint32
		Ports                []Port
		Vrfs                 []*Vrf
		FRRConfig            *FRRConfig
		MetalCoreCIDR        string
		AdditionalMgmtRoutes []string
		Eth0                 MgmtInterface
		PXEVlanID            uint16
		SetSrcLoopback       bool
	}

	Port struct {
		Name        string
		AdminStatus PortStatus
		VLAN        *string
		VRF         *string
		MTU         uint
	}

	Vrf struct {
		Name   string
		VNI    uint32
		VLANID uint16
	}

	FRRConfig struct {
		VRFs                 []string
		UnnumberedInterfaces []*Interface
		BGP                  []*BGP
		RouteMaps            []*RouteMap
		PrefixLists          []*IPPrefixList
	}

	Interface struct {
		Name string
		Vrf  *string
	}

	BGP struct {
		Vrf             *string
		PeerGroups      []PeerGroup
		Neighbors       []BGPNeighbor
		AddressFamilies BGPAddressFamilies
	}

	BGPNeighbor struct {
		Name      string
		PeerGroup *string
	}

	BGPAddressFamily struct {
		GlobalStatements []string
		Neighbors        []AdressFamilyNeighbor
	}

	BGPAddressFamilies struct {
		IPv4Unicast *BGPAddressFamily
		IPv6Unicast *BGPAddressFamily
		L2VPNEVPN   *BGPAddressFamily
	}

	// FIXME: naming
	AdressFamilyNeighbor struct {
		Neighbor    string
		Commands    []string
		RouteMapIn  string
		RouteMapOut string
	}

	PeerGroup struct {
		Name     string
		Commands []string
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

// func (s *Filter) Assemble(rmPrefix string, vnis, cidrs []string) {
// 	cidrsByAf := cidrsByAddressfamily(cidrs)
// 	if len(cidrsByAf.ipv4Cidrs) > 0 {
// 		prefixRouteMapName := fmt.Sprintf("%s-in", rmPrefix)
// 		prefixListName := fmt.Sprintf("%s-in-prefixes", rmPrefix)
// 		rm := RouteMap{
// 			Name:    prefixRouteMapName,
// 			Entries: []string{fmt.Sprintf("match ip address prefix-list %s", prefixListName)},
// 			Policy:  PolicyPermit,
// 			Order:   10,
// 		}
// 		s.RouteMaps = append(s.RouteMaps, rm)
// 		s.addPrefixList(prefixListName, cidrsByAf.ipv4Cidrs, AddressFamilyIPv4)
// 	}
// 	if len(cidrsByAf.ipv6Cidrs) > 0 {
// 		prefixRouteMapName := fmt.Sprintf("%s-in6", rmPrefix)
// 		prefixListName := fmt.Sprintf("%s-in6-prefixes", rmPrefix)
// 		rm := RouteMap{
// 			Name:    prefixRouteMapName,
// 			Entries: []string{fmt.Sprintf("match ipv6 address prefix-list %s", prefixListName)},
// 			Policy:  PolicyPermit,
// 			Order:   10,
// 		}
// 		s.RouteMaps = append(s.RouteMaps, rm)
// 		s.addPrefixList(prefixListName, cidrsByAf.ipv6Cidrs, AddressFamilyIPv6)
// 	}
// 	if len(vnis) > 0 {
// 		vniRouteMapName := fmt.Sprintf("%s-vni", rmPrefix)
// 		for j, vni := range vnis {
// 			rm := RouteMap{
// 				Name:    vniRouteMapName,
// 				Entries: []string{fmt.Sprintf("match evpn vni %s", vni)},
// 				Policy:  PolicyPermit,
// 				Order:   10 + j,
// 			}
// 			s.RouteMaps = append(s.RouteMaps, rm)
// 		}
// 	}
// }

// func (s *Filter) addPrefixList(prefixListName string, cidrs []string, af AddressFamily) {
// 	for _, cidr := range cidrs {
// 		prefix, err := netip.ParsePrefix(cidr)
// 		if err != nil {
// 			continue
// 		}
// 		spec := fmt.Sprintf("permit %s le %d", cidr, prefix.Addr().BitLen())
// 		prefixList := IPPrefixList{
// 			AddressFamily: af,
// 			Name:          prefixListName,
// 			Spec:          spec,
// 		}
// 		s.IPPrefixLists = append(s.IPPrefixLists, prefixList)
// 	}
// }

// func (p *Port) HasAddressFamilies() (has4 bool, has6 bool, err error) {
// 	for _, cidr := range p.Cidrs {
// 		addr, _, err := net.ParseCIDR(cidr)
// 		if err != nil {
// 			return false, false, err
// 		}
// 		if addr.To4() != nil {
// 			has4 = true
// 		} else {
// 			has6 = true
// 		}
// 	}

// 	return has4, has6, nil
// }

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
