package types

import "fmt"

// Conf holds the switch configuration
// nolint:musttag
type Conf struct {
	Name                 string
	LogLevel             string
	Loopback             string
	ASN                  uint32
	Ports                Ports
	MetalCoreCIDR        string
	AdditionalBridgeVIDs []string
	DHCPServers          []string
}

type Ports struct {
	Eth0          Nic
	Underlay      []string
	Provisioned   []string
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
	if len(cidrs) > 0 {
		prefixRouteMapName := fmt.Sprintf("%s-in", rmPrefix)
		prefixListName := fmt.Sprintf("%s-in-prefixes", rmPrefix)
		rm := RouteMap{
			Name:    prefixRouteMapName,
			Entries: []string{fmt.Sprintf("match ip address prefix-list %s", prefixListName)},
			Policy:  "permit",
			Order:   10,
		}
		s.RouteMaps = append(s.RouteMaps, rm)

		for _, cidr := range cidrs {
			spec := fmt.Sprintf("permit %s le 32", cidr)
			prefixList := IPPrefixList{
				Name: prefixListName,
				Spec: spec,
			}
			s.IPPrefixLists = append(s.IPPrefixLists, prefixList)
		}
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
