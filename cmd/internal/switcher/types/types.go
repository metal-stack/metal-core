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
		MetalCoreCIDR        string
		AdditionalBridgeVIDs []string
		AdditionalMgmtRoutes []string
		PXEVlanID            uint16
		SetSrcLoopback       bool
	}

	Ports struct {
		Eth0          Nic
		Underlay      []string
		Unprovisioned []string
		BladePorts    []string
		Vrfs          Vrfs
		Firewalls     map[string]*Firewall
		AdminStatus   map[string]PortStatus
	}

	StaticBGPConfig struct {
		BGPSessions []BGPSession                    `yaml:"bgp_sessions"`
		PeerGroups  map[string]map[string]PeerGroup `yaml:"peer_groups"`
	}

	BGPSession struct {
		VRF       string             `yaml:"vrf"`
		Neighbors map[string]BGPPeer `yaml:"neighbors"`
	}

	BGPPeer struct {
		PeerGroup  string `yaml:"peer_group"`
		Unnumbered bool   `yaml:"-"`
	}

	PeerGroup struct {
		Cidrs  []string `yaml:"cidrs"`
		Filter Filter   `yaml:"filter"`
		// TODO: document in the README that activating EVPN will only work for peers in the default VRF
		// That's because only one VRF can be the underlay for sharing EVPN routes and there currently is no need for allowing to change that to something other than the default VRF.
		EVPN     bool     `yaml:"evpn"`
		Commands []string `yaml:"commands"`
	}

	Vrfs map[string]*Vrf

	Vrf struct {
		Filter
		VNI       uint32
		VLANID    uint16
		Neighbors []string
		Cidrs     []string
		Has4      bool
		Has6      bool
	}

	Firewall struct {
		Filter
		Port  string
		Cidrs []string
		Vnis  []string
	}

	Filter struct {
		IPPrefixLists []IPPrefixList `yaml:"ip_prefix_lists"`
		RouteMaps     []RouteMap     `yaml:"route_maps"`
	}

	Nic struct {
		AddressCIDR string
		Gateway     string
	}

	RouteMap struct {
		Name    string         `yaml:"name"`
		Entries []string       `yaml:"entries"`
		Policy  RouteMapPolicy `yaml:"policy"`
		Order   int            `yaml:"order"`
	}

	IPPrefixList struct {
		AddressFamily AddressFamily `yaml:"address_family"`
		Name          string        `yaml:"name"`
		Spec          string        `yaml:"spec"`
	}

	cidrsByAf struct {
		ipv4Cidrs []string
		ipv6Cidrs []string
	}

	PortStatus     string
	AddressFamily  string
	RouteMapPolicy string
)

const (
	PortStatusUp   = PortStatus("up")
	PortStatusDown = PortStatus("down")

	AddressFamilyIPv4 = AddressFamily("ip")
	AddressFamilyIPv6 = AddressFamily("ipv6")

	RouteMapPolicyPermit = RouteMapPolicy("permit")
	RouteMapPolicyDeny   = RouteMapPolicy("deny")
)

func (s *Filter) Assemble(rmPrefix string, vnis, cidrs []string) {
	cidrsByAf := cidrsByAddressfamily(cidrs)
	if len(cidrsByAf.ipv4Cidrs) > 0 {
		prefixRouteMapName := fmt.Sprintf("%s-in", rmPrefix)
		prefixListName := fmt.Sprintf("%s-in-prefixes", rmPrefix)
		rm := RouteMap{
			Name:    prefixRouteMapName,
			Entries: []string{fmt.Sprintf("match ip address prefix-list %s", prefixListName)},
			Policy:  RouteMapPolicyPermit,
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
			Policy:  RouteMapPolicyPermit,
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
				Policy:  RouteMapPolicyPermit,
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
