package core

import (
	"log/slog"
	"os"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const (
	testcoreCIDR           = "10.255.0.1/24"
	testcoreLoopback       = "10.1.1.1"
	testcoreASN            = "42"
	testcoreMgmtGateway    = "10.253.1.129/30"
	testcorePXEVLAN        = 4000
	testcoreSetSrcLoopback = true
)

var (
	testcoreAdditionalMgmtRoutes  = []string{}
	testcoreAdditionalBridgePorts = []string{}
	testcoreAdditionalBridgeVIDs  = []string{}
	testcoreSpineUplinks          = []string{"Ethernet120", "Ethernet124"}
	testcoreLog                   = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	testcoreCfg = Config{
		Log:                   testcoreLog,
		CIDR:                  testcoreCIDR,
		LoopbackIP:            testcoreLoopback,
		ASN:                   testcoreASN,
		ManagementGateway:     testcoreMgmtGateway,
		PXEVlanID:             testcorePXEVLAN,
		AdditionalMgmtRoutes:  testcoreAdditionalMgmtRoutes,
		AdditionalBridgePorts: testcoreAdditionalBridgePorts,
		AdditionalBridgeVIDs:  testcoreAdditionalBridgeVIDs,
		SpineUplinks:          testcoreSpineUplinks,
		SetSrcLoopback:        testcoreSetSrcLoopback,
		NOS:                   &sonic.Sonic{},
	}

	leaf01 = &apiv2.Switch{
		Id: "leaf01",
		Nics: []*apiv2.SwitchNic{
			{
				Name:       "Ethernet0",
				Identifier: "Eth1/1",
			},
			{
				Name:       "Ethernet1",
				Identifier: "Eth1/2",
				State: &apiv2.NicState{
					Desired: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN.Enum(),
					Actual:  apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP,
				},
			},
			{
				Name:       "Ethernet2",
				Identifier: "Eth1/3",
			},
			{
				Name:       "Ethernet3",
				Identifier: "Eth1/4",
			},
			{
				Name:       "Ethernet4",
				Identifier: "Eth2/1",
				Vrf:        new("default"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{"10.32.24.0/24"},
					Vnis:  []string{"104009", "200"},
				},
			},
			{
				Name:       "Ethernet5",
				Identifier: "Eth2/2",
				Vrf:        new("vrf200"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{"10.32.24.0/24"},
				},
			},
			{
				Name:       "Ethernet6",
				Identifier: "Eth2/3",
				State: &apiv2.NicState{
					Desired: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN.Enum(),
					Actual:  apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP,
				},
				Vrf: new("vrf100"),
				BgpFilter: &apiv2.BGPFilter{
					Cidrs: []string{"10.32.25.0/24"},
				},
			},
			{
				Name:       "Ethernet120",
				Identifier: "Eth31",
			},
			{
				Name:       "Ethernet124",
				Identifier: "Eth32",
			},
		},
	}

	leaf01PreviousCfg = &types.Conf{
		Ports: types.Ports{
			Unprovisioned: []string{"Ethernet0", "Ethernet1", "Ethernet4", "Ethernet5"},
			Underlay:      []string{"Ethernet120", "Ethernet124"},
			Vrfs: types.Vrfs{
				"Vrf300": {
					VNI:       300,
					VLANID:    1003,
					Neighbors: []string{"Ethernet3"},
				},
				"Vrf100": {
					VNI:       100,
					VLANID:    1001,
					Neighbors: []string{"Ethernet6"},
				},
			},
			Firewalls: map[string]*types.Firewall{
				"Ethernet2": {
					Port: "Ethernet2",
				},
			},
			AdminStatus: map[string]types.PortStatus{
				"Ethernet0":   types.PortStatusUp,
				"Ethernet1":   types.PortStatusUp,
				"Ethernet2":   types.PortStatusUp,
				"Ethernet3":   types.PortStatusUp,
				"Ethernet4":   types.PortStatusUp,
				"Ethernet5":   types.PortStatusUp,
				"Ethernet6":   types.PortStatusUp,
				"Ethernet120": types.PortStatusUp,
				"Ethernet124": types.PortStatusUp,
			},
		},
		Name:                 leaf01.Id,
		Loopback:             testcoreLoopback,
		MetalCoreCIDR:        testcoreCIDR,
		AdditionalBridgeVIDs: testcoreAdditionalBridgeVIDs,
		AdditionalMgmtRoutes: testcoreAdditionalMgmtRoutes,
		PXEVlanID:            testcorePXEVLAN,
		SetSrcLoopback:       testcoreSetSrcLoopback,
	}

	leaf01PreviousConfigDB = test.StringMap{
		"INTERFACE": test.StringMap{
			"Ethernet120": test.StringMap{
				"ipv6_use_link_local_only": "enable",
			},
			"Ethernet124": test.StringMap{
				"ipv6_use_link_local_only": "enable",
			},
			"Ethernet2": test.StringMap{
				"ipv6_use_link_local_only": "enable",
			},
			"Ethernet3": test.StringMap{
				"ipv6_use_link_local_only": "enable",
				"vrf_name":                 "Vrf300",
			},
			"Ethernet6": test.StringMap{
				"ipv6_use_link_local_only": "enable",
				"vrf_name":                 "Vrf100",
			},
		},
		"PORT": test.StringMap{
			"Ethernet0": test.StringMap{
				"admin_status": "up",
			},
			"Ethernet1": test.StringMap{
				"admin_status": "up",
			},
			"Ethernet2": test.StringMap{
				"admin_status": "up",
			},
			"Ethernet3": test.StringMap{
				"admin_status": "up",
			},
			"Ethernet4": test.StringMap{
				"admin_status": "up",
			},
			"Ethernet5": test.StringMap{
				"admin_status": "up",
			},
			"Ethernet6": test.StringMap{
				"admin_status": "up",
			},
			"Ethernet120": test.StringMap{
				"admin_status": "up",
			},
			"Ethernet124": test.StringMap{
				"admin_status": "up",
			},
		},
		"SUPPRESS_VLAN_NEIGH": test.StringMap{
			"Vlan1001": test.StringMap{
				"suppress": "on",
			},
			"Vlan1003": test.StringMap{
				"suppress": "on",
			},
		},
		"VLAN_INTERFACE": test.StringMap{
			"Vlan1001": test.StringMap{
				"vrf_name": "Vrf100",
			},
			"Vlan1003": test.StringMap{
				"vrf_name": "Vrf300",
			},
		},
		"VLAN_MEMBER": test.StringMap{
			"Vlan4000": test.StringMap{
				"Ethernet0": test.StringMap{
					"tagging_mode": "untagged",
				},
				"Ethernet1": test.StringMap{
					"tagging_mode": "untagged",
				},
				"Ethernet4": test.StringMap{
					"tagging_mode": "untagged",
				},
				"Ethernet5": test.StringMap{
					"tagging_mode": "untagged",
				},
			},
		},
		"VLAN": test.StringMap{
			"Vlan1001": test.StringMap{
				"vlanid": "1001",
			},
			"Vlan1003": test.StringMap{
				"vlanid": "1003",
			},
			"Vlan4000": test.StringMap{
				"vlanid": "4000",
			},
		},
		"VRF": test.StringMap{
			"Vrf100": test.StringMap{
				"fallback": "false",
				"vni":      "100",
			},
			"Vrf300": test.StringMap{
				"fallback": "false",
				"vni":      "300",
			},
		},
		"VXLAN_TUNNEL_MAP": test.StringMap{
			"vtep": test.StringMap{
				"map_100_Vlan1001": test.StringMap{
					"vlan": "Vlan1001",
					"vni":  "100",
				},
				"map_300_Vlan1003": test.StringMap{
					"vlan": "Vlan1003",
					"vni":  "300",
				},
			},
		},
		"VXLAN_TUNNEL": test.StringMap{
			"vtep": test.StringMap{
				"src_ip": testcoreCfg.LoopbackIP,
			},
		},
	}

	leaf01WantConfigDB = test.HashMap{
		"INTERFACE|Ethernet120": map[string]string{
			"ipv6_use_link_local_only": "enable",
		},
		"INTERFACE|Ethernet124": map[string]string{
			"ipv6_use_link_local_only": "enable",
		},
		"INTERFACE|Ethernet4": map[string]string{
			"ipv6_use_link_local_only": "enable",
		},
		"INTERFACE|Ethernet5": map[string]string{
			"ipv6_use_link_local_only": "enable",
			"vrf_name":                 "Vrf200",
		},
		"INTERFACE|Ethernet6": map[string]string{
			"ipv6_use_link_local_only": "enable",
			"vrf_name":                 "Vrf100",
		},
		"PORT|Ethernet0": map[string]string{
			"admin_status": "up",
			"mtu":          "9000",
		},
		"PORT|Ethernet1": map[string]string{
			"admin_status": "down",
			"mtu":          "9000",
		},
		"PORT|Ethernet2": map[string]string{
			"admin_status": "up",
			"mtu":          "9000",
		},
		"PORT|Ethernet3": map[string]string{
			"admin_status": "up",
			"mtu":          "9000",
		},
		"PORT|Ethernet4": map[string]string{
			"admin_status": "up",
			"mtu":          "9216",
		},
		"PORT|Ethernet5": map[string]string{
			"admin_status": "up",
			"mtu":          "9000",
		},
		"PORT|Ethernet6": map[string]string{
			"admin_status": "down",
			"mtu":          "9000",
		},
		"PORT|Ethernet120": map[string]string{
			"admin_status": "up",
			"mtu":          "9216",
		},
		"PORT|Ethernet124": map[string]string{
			"admin_status": "up",
			"mtu":          "9216",
		},
		"SUPPRESS_VLAN_NEIGH|Vlan1001": map[string]string{
			"suppress": "on",
		},
		"SUPPRESS_VLAN_NEIGH|Vlan1002": map[string]string{
			"suppress": "on",
		},
		"VLAN_INTERFACE|Vlan1001": map[string]string{
			"vrf_name": "Vrf100",
		},
		"VLAN_INTERFACE|Vlan1002": map[string]string{
			"vrf_name": "Vrf200",
		},
		"VLAN_MEMBER|Vlan4000|Ethernet0": map[string]string{
			"tagging_mode": "untagged",
		},
		"VLAN_MEMBER|Vlan4000|Ethernet1": map[string]string{
			"tagging_mode": "untagged",
		},
		"VLAN_MEMBER|Vlan4000|Ethernet2": map[string]string{
			"tagging_mode": "untagged",
		},
		"VLAN_MEMBER|Vlan4000|Ethernet3": map[string]string{
			"tagging_mode": "untagged",
		},
		"VLAN|Vlan1001": map[string]string{
			"vlanid": "1001",
		},
		"VLAN|Vlan1002": map[string]string{
			"vlanid": "1002",
		},
		"VLAN|Vlan4000": map[string]string{
			"vlanid": "4000",
		},
		"VRF|Vrf100": map[string]string{
			"fallback": "false",
			"vni":      "100",
		},
		"VRF|Vrf200": map[string]string{
			"fallback": "false",
			"vni":      "200",
		},
		"VXLAN_TUNNEL_MAP|vtep|map_100_Vlan1001": map[string]string{
			"vlan": "Vlan1001",
			"vni":  "100",
		},
		"VXLAN_TUNNEL_MAP|vtep|map_200_Vlan1002": map[string]string{
			"vlan": "Vlan1002",
			"vni":  "200",
		},
		"VXLAN_TUNNEL|vtep": map[string]string{
			"src_ip": testcoreCfg.LoopbackIP,
		},
	}
)

func injectStaticVrfVlanMapping(conf *types.Conf, mapping map[string]uint16) {
	for vrf, vlan := range mapping {
		conf.Ports.Vrfs[vrf].VLANID = vlan
	}
}
