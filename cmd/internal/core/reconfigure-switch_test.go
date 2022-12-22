package core

import (
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/cumulus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"

	"github.com/metal-stack/metal-go/api/models"
	"github.com/stretchr/testify/require"
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
	swp1 := models.V1SwitchNic{
		Name: &n1,
		Mac:  &m1,
	}
	n2 := "swp2"
	m2 := "00:00:00:00:00:02"
	swp2 := models.V1SwitchNic{
		Name: &n2,
		Mac:  &m2,
		Vrf:  "vrf104001",
	}
	n3 := "swp3"
	m3 := "00:00:00:00:00:03"
	swp3 := models.V1SwitchNic{
		Name: &n3,
		Mac:  &m3,
		Vrf:  "default",
	}
	s := &models.V1SwitchResponse{
		Nics: []*models.V1SwitchNic{
			&swp1,
			&swp2,
			&swp3,
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
							Name: "vrf104001-in-prefixes",
							Spec: "permit 10.244.0.0/16 le 32",
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
				Cidrs: []string{"10.244.0.0/16"},
			}},
		},
		AdditionalBridgeVIDs: []string{"201-256", "301-356"},
	}
	require.EqualValues(t, expected, actual)
}
