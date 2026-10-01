package core

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func TestBuildSwitcherConfig(t *testing.T) {
	c := &Core{
		cidr:                 "10.255.255.2/24",
		partitionID:          "partition1",
		rackID:               "rack01",
		roomID:               "room01",
		asn:                  "420000001",
		loopbackIP:           "10.0.0.1",
		spineUplinks:         []string{"Ethernet120", "Ethernet124"},
		additionalBridgeVIDs: []string{"201-256", "301-356"},
		nos:                  &sonic.Sonic{},
		staticVRFs: types.Vrfs{
			"vrf200": {
				VNI:       200,
				Neighbors: []string{"Ethernet3"},
				Cidrs:     []string{"10.1.2.0/24"},
			},
		},
	}

	s := &apiv2.Switch{
		Nics: []*apiv2.SwitchNic{
			{
				Name: "Ethernet0",
			},
			{
				Name: "Ethernet1",
				Vrf:  new("vrf104001"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{
						"10.240.0.0/12",
					},
				},
			},
			{
				Name: "Ethernet2",
				Vrf:  new("default"),
			},
			{
				Name: "Ethernet3",
			},
			{
				Name: "Ethernet120",
			},
			{
				Name: "Ethernet124",
			},
		},
	}

	actual, err := c.buildSwitcherConfig(s)
	require.NoError(t, err)

	expected := &types.Conf{
		LogLevel:      "warnings",
		Loopback:      "10.0.0.1",
		MetalCoreCIDR: "10.255.255.2/24",
		ASN:           420000001,
		Ports: types.Ports{
			AdminStatus:   map[string]types.PortStatus{},
			Underlay:      []string{"Ethernet120", "Ethernet124"},
			Unprovisioned: []string{"Ethernet0"},
			Firewalls: map[string]*types.Firewall{
				"Ethernet2": {
					Port: "Ethernet2",
				},
			},
			Vrfs: map[string]*types.Vrf{
				"Vrf200": {
					VNI:       200,
					Neighbors: []string{"Ethernet3"},
					Cidrs:     []string{"10.1.2.0/24"},
					Filter: types.Filter{
						IPPrefixLists: []types.IPPrefixList{
							{
								AddressFamily: "ip",
								Name:          "Vrf200-in-prefixes",
								Spec:          "permit 10.1.2.0/24 le 32",
							},
						},
						RouteMaps: []types.RouteMap{
							{
								Name:    "Vrf200-in",
								Entries: []string{"match ip address prefix-list Vrf200-in-prefixes"},
								Policy:  "permit",
								Order:   10,
							},
						},
					},
					Has4: true,
				},
				"Vrf104001": {
					VNI:       104001,
					Neighbors: []string{"Ethernet1"},
					Filter: types.Filter{
						IPPrefixLists: []types.IPPrefixList{
							{
								AddressFamily: "ip",
								Name:          "Vrf104001-in-prefixes",
								Spec:          "permit 10.240.0.0/12 le 32",
							},
						},
						RouteMaps: []types.RouteMap{
							{
								Name:    "Vrf104001-in",
								Entries: []string{"match ip address prefix-list Vrf104001-in-prefixes"},
								Policy:  "permit",
								Order:   10,
							},
						},
					},
					Cidrs: []string{"10.240.0.0/12"},
					Has4:  true,
				},
			},
		},
		AdditionalBridgeVIDs: []string{"201-256", "301-356"},
	}
	// VLANID is ignored because it is reserved dynamically and the order in which the map is iterated matters so the test becomes flaky.
	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreFields(types.Vrf{}, "VLANID")); diff != "" {
		t.Errorf("TestBuildSwitcherConfig() diff = %s", diff)
	}

	expectedStaticVrfs := types.Vrfs{
		"vrf200": {
			VNI:       200,
			Neighbors: []string{"Ethernet3"},
			Cidrs:     []string{"10.1.2.0/24"},
		},
	}
	if diff := cmp.Diff(expectedStaticVrfs, c.staticVRFs); diff != "" {
		t.Errorf("TestBuildSwitcherConfig() Core.staticVrfs unexpectedly changed: %s", diff)
	}
}
