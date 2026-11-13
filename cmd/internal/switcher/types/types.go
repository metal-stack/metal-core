package types

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
