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

	Vrfs map[string]*Vrf

	Vrf struct {
		Filter    `yaml:"filter"`
		VNI       uint32   `yaml:"vni"`
		VLANID    uint16   `yaml:"vlanid"`
		Neighbors []string `yaml:"neighbors"`
		Cidrs     []string `yaml:"cidrs"`
		Has4      bool     `yaml:"has4"`
		Has6      bool     `yaml:"has6"`
	}

	Firewall struct {
		Filter `yaml:"filter"`
		Port   string   `yaml:"port"`
		Cidrs  []string `yaml:"cidrs"`
		Vnis   []string `yaml:"vnis"`
	}

	Filter struct {
		IPPrefixLists []IPPrefixList `yaml:"ip-prefix-lists"`
		RouteMaps     []RouteMap     `yaml:"route-maps"`
	}

	Nic struct {
		AddressCIDR string
		Gateway     string
	}

	RouteMap struct {
		Name    string   `yaml:"name"`
		Entries []string `yaml:"entries"`
		Policy  string   `yaml:"policy"`
		Order   int      `yaml:"order"`
	}

	IPPrefixList struct {
		AddressFamily string `yaml:"address-family"`
		Name          string `yaml:"name"`
		Spec          string `yaml:"spec"`
	}

	PortStatus string
)

const (
	PortStatusUp   = PortStatus("up")
	PortStatusDown = PortStatus("down")
)

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
