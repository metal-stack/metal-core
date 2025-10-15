package core

import (
	"testing"

	"github.com/stretchr/testify/require"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func TestBuildSwitcherConfig(t *testing.T) {
	c := &Core{
		cidr:                 "10.255.255.2/24",
		partitionID:          "fra-equ01",
		rackID:               "rack01",
		asn:                  "420000001",
		loopbackIP:           "10.0.0.1",
		spineUplinks:         []string{"swp31", "swp32"},
		additionalBridgeVIDs: []string{"201-256", "301-356"},
		nos:                  &cumulus.Cumulus{},
	}

	n1 := "swp1"
	m1 := "00:00:00:00:00:01"
	swp1 := &apiv2.SwitchNic{
		Name: n1,
		Mac:  m1,
	}
	n2 := "swp2"
	m2 := "00:00:00:00:00:02"
	swp2 := &apiv2.SwitchNic{
		Name: n2,
		Mac:  m2,
		Vrf:  pointer.Pointer("vrf104001"),
		BgpFilter: &apiv2.BGPFilter{
			Cidrs: []string{
				"10.240.0.0/12", // pod ipv4 cidrs
			},
		},
	}
	n3 := "swp3"
	m3 := "00:00:00:00:00:03"
	swp3 := &apiv2.SwitchNic{
		Name: n3,
		Mac:  m3,
		Vrf:  pointer.Pointer("default"),
	}
	s := &apiv2.Switch{
		Nics: []*apiv2.SwitchNic{
			swp1,
			swp2,
			swp3,
		},
	}
	actual, err := c.buildSwitcherConfig(s)
	require.NoError(t, err)
	require.NotNil(t, actual)
	expected := &types.Conf{
		LogLevel:      "warnings",
		Loopback:      "10.0.0.1",
		MetalCoreCIDR: "10.255.255.2/24",
		ASN:           420000001,
		Ports: types.Ports{
			DownPorts:     map[string]bool{},
			Underlay:      []string{"swp31", "swp32"},
			Unprovisioned: []string{"swp1"},
			Firewalls: map[string]*types.Firewall{
				"swp3": {
					Port: "swp3",
				},
			},
			Vrfs: map[string]*types.Vrf{"vrf104001": {
				VNI:       104001,
				VLANID:    1001,
				Neighbors: []string{"swp2"},
				Filter: types.Filter{
					IPPrefixLists: []types.IPPrefixList{
						{
							AddressFamily: "ip",
							Name:          "vrf104001-in-prefixes",
							Spec:          "permit 10.240.0.0/12 le 32",
						},
					},
					RouteMaps: []types.RouteMap{
						{
							Name:    "vrf104001-in",
							Entries: []string{"match ip address prefix-list vrf104001-in-prefixes"},
							Policy:  "permit",
							Order:   10,
						},
					},
				},
				Cidrs: []string{"10.240.0.0/12"},
				Has4:  true,
			}},
		},
		AdditionalBridgeVIDs: []string{"201-256", "301-356"},
	}
	require.EqualValues(t, expected, actual)
}
