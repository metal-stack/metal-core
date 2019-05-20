package event

import (
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metal/metal-core/switcher"
	"github.com/stretchr/testify/require"
)

func TestBuildSwitcherConfig(t *testing.T) {
	config := &domain.Config{
		CIDR:                 "10.255.255.2/24",
		PartitionID:          "fra-equ01",
		RackID:               "rack01",
		ASN:                  "420000001",
		LoopbackIP:           "10.0.0.1",
		SpineUplinks:         "swp31,swp32",
		AdditionalBridgeVIDs: []string{"201-256", "301-356"},
	}
	n1 := "swp1"
	m1 := "00:00:00:00:00:01"
	swp1 := models.MetalNic{
		Name: &n1,
		Mac:  &m1,
	}
	n2 := "swp2"
	m2 := "00:00:00:00:00:02"
	swp2 := models.MetalNic{
		Name: &n2,
		Mac:  &m2,
		Vrf:  "vrf104001",
	}
	s := &models.MetalSwitch{
		Nics: []*models.MetalNic{
			&swp1,
			&swp2,
		},
	}
	actual, err := buildSwitcherConfig(config, s)
	require.NoError(t, err)
	require.NotNil(t, actual)
	expected := &switcher.Conf{
		Loopback:      "10.0.0.1",
		MetalCoreCIDR: "10.255.255.2/24",
		ASN:           420000001,
		Neighbors:     []string{"swp31", "swp32"},
		Unprovisioned: []string{"swp1"},
		Tenants: map[string]*switcher.Tenant{"vrf104001": {
			VNI:       104001,
			VLANID:    1001,
			Neighbors: []string{"swp2"},
		}},
		AdditionalBridgeVIDs: []string{"201-256", "301-356"},
	}
	require.EqualValues(t, expected, actual)
}
