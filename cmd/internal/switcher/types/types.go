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
		// Boot holds the configuration of the boot vrf (MEP-20). If nil, unprovisioned ports are put into the PXE vlan.
		Boot *BootConf
	}

	// BootConf describes the layer 3 boot vrf for unprovisioned ports (MEP-20).
	// Every unprovisioned port becomes a routed interface in this vrf with its own /64 prefix
	// from which booting machines derive their address via SLAAC.
	BootConf struct {
		// Vrf is the name of the boot vrf.
		Vrf string
		// VNI is the layer 3 vni of the boot vrf.
		VNI uint32
		// VLANID is the switch-local vlan that maps the vni, filled by FillVLANIDs.
		VLANID uint16
		// RDNSS are the recursive dns server addresses that are advertised to booting machines.
		RDNSS []string
		// Ports maps every unprovisioned port to its boot prefix.
		Ports map[string]BootPort
	}

	// BootPort holds the per port boot prefix.
	BootPort struct {
		// Prefix is the /64 that is advertised on the port, e.g. fd00:20:0:100::/64
		Prefix string
		// Address is the address of the switch inside the prefix, e.g. fd00:20:0:100::1/64
		Address string
	}

	Ports struct {
		Eth0          Nic
		Underlay      []string
		Unprovisioned []string
		BladePorts    []string
		Vrfs          map[string]*Vrf
		Firewalls     map[string]*Firewall
		AdminStatus   map[string]PortStatus
	}

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
		IPPrefixLists []IPPrefixList
		RouteMaps     []RouteMap
	}

	Nic struct {
		AddressCIDR string
		Gateway     string
	}

	RouteMap struct {
		Name    string
		Entries []string
		Policy  string
		Order   int
	}

	IPPrefixList struct {
		AddressFamily string
		Name          string
		Spec          string
	}

	cidrsByAf struct {
		ipv4Cidrs []string
		ipv6Cidrs []string
	}

	PortStatus string
)

const (
	PortStatusUp   = PortStatus("up")
	PortStatusDown = PortStatus("down")

	// BootVrfName is the fixed name of the boot vrf on a switch.
	BootVrfName = "VrfBoot"
	// BootPrefixLength is the prefix length of every per port boot prefix.
	BootPrefixLength = 64
	// MaxVNI is the largest possible vxlan network identifier (24 bit).
	MaxVNI = 1<<24 - 1
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
