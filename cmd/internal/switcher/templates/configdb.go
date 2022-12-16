package templates

import (
	"encoding/json"
	"fmt"
	"github.com/coreos/go-systemd/v22/unit"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"os"
	"strconv"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
)

const (
	// Tried to use "metal-core" name for the file. It doesn't work.
	// Systemd transforms "-" to "\" when %I specifier is used.
	metalCoreConfigdb     = "/etc/sonic/metal.json"
	metalCoreConfigdbTmp  = "/etc/sonic/metal.tmp"
	configdbReloadService = "write-to-db"

	untagged = "untagged"
)

type configdb struct {
	Features       map[string]*feature        `json:"FEATURE"`
	Loopback       map[string]struct{}        `json:"LOOPBACK_INTERFACE"`
	Ifaces         map[string]*iface          `json:"INTERFACE"`
	Ports          map[string]*port           `json:"PORT"`
	Vlans          map[string]*vlan2          `json:"VLAN"`
	VlanIfaces     map[string]*iface          `json:"VLAN_INTERFACE"`
	VlanMembers    map[string]*vlanMember     `json:"VLAN_MEMBER"`
	Vrfs           map[string]*vrf            `json:"VRF"`
	VxlanTunnel    vxlanTunnel                `json:"VXLAN_TUNNEL"`
	VxlanTunnelMap map[string]*vxlanTunnelMap `json:"VXLAN_TUNNEL_MAP"`
}

type feature struct {
	State string `json:"state,omitempty"`
}

type iface struct {
	VrfName string `json:"vrf_name,omitempty"`
}

type port struct {
	Mtu string `json:"mtu,omitempty"`
}

type vlan2 struct {
	VlanId      string   `json:"vlanid,omitempty"`
	DHCPServers []string `json:"dhcp_servers,omitempty"`
}

type vlanMember struct {
	TaggingMode string `json:"tagging_mode"`
}

type vrf struct {
	Vni string `json:"vni,omitempty"`
}

type vxlanTunnel struct {
	Vtep vtep `json:"vtep"`
}

type vtep struct {
	SrcIp string `json:"src_ip"`
}

type vxlanTunnelMap struct {
	Vlan string `json:"vlan"`
	Vni  string `json:"vni"`
}

func buildConfigdb(cfg *types.Conf, fpInfs []string) *configdb {
	c := &configdb{
		Features:       map[string]*feature{},
		Loopback:       map[string]struct{}{},
		Ifaces:         map[string]*iface{},
		Ports:          map[string]*port{},
		Vlans:          map[string]*vlan2{},
		VlanIfaces:     map[string]*iface{},
		VlanMembers:    map[string]*vlanMember{},
		Vrfs:           map[string]*vrf{},
		VxlanTunnel:    vxlanTunnel{vtep{SrcIp: cfg.Loopback}},
		VxlanTunnelMap: map[string]*vxlanTunnelMap{},
	}

	c.Features["dhcp_relay"] = &feature{
		State: "enabled",
	}

	c.Loopback["Loopback0"] = struct{}{}
	if cfg.Loopback != "" {
		c.Loopback[fmt.Sprintf("Loopback0|%s/32", cfg.Loopback)] = struct{}{}
	}

	for _, p := range cfg.Ports.Underlay {
		c.Ifaces[p] = &iface{}
		c.Ports[p] = &port{Mtu: "9216"}
	}
	for _, fw := range cfg.Ports.Firewalls {
		c.Ifaces[fw.Port] = &iface{}
		c.Ports[fw.Port] = &port{Mtu: "9216"}
	}
	for vrfName, v := range cfg.Ports.Vrfs {
		for _, p := range v.Neighbors {
			c.Ifaces[p] = &iface{vrfName}
			c.Ports[p] = &port{Mtu: "9000"}
		}
		vlanId := strconv.FormatUint(uint64(v.VLANID), 10)
		vlanName := "Vlan" + vlanId
		vni := strconv.FormatUint(uint64(v.VNI), 10)
		c.Vlans[vlanName] = &vlan2{VlanId: vlanId}
		c.VlanIfaces[vlanName] = &iface{vrfName}
		c.Vrfs[vrfName] = &vrf{vni}

		tunnelMapName := fmt.Sprintf("vtep|map_%s_%s", vni, vlanName)
		c.VxlanTunnelMap[tunnelMapName] = &vxlanTunnelMap{vlanName, vni}
	}
	pxeIfaceName := "Vlan4000|" + cfg.MetalCoreCIDR
	c.Vlans["Vlan4000"] = &vlan2{VlanId: "4000", DHCPServers: cfg.DHCPServers}
	c.VlanIfaces["Vlan4000"] = &iface{}
	c.VlanIfaces[pxeIfaceName] = &iface{}
	c.VxlanTunnelMap["vtep|map_104000_Vlan4000"] = &vxlanTunnelMap{"Vlan4000", "104000"}
	for _, p := range cfg.Ports.Unprovisioned {
		memberName := "Vlan4000|" + p
		c.Ifaces[p] = &iface{}
		c.Ports[p] = &port{Mtu: "9000"}
		c.VlanMembers[memberName] = &vlanMember{untagged}
	}

	// remove IPs for front-panel interfaces
	for _, inf := range fpInfs {
		if strings.Contains(inf, "|") {
			c.Ifaces[inf] = nil
		}
	}

	return c
}

type ConfigdbApplier struct {
	interfaces []string
}

func NewConfigdbApplier(infs []string) *ConfigdbApplier {
	return &ConfigdbApplier{
		interfaces: infs,
	}
}

func (a *ConfigdbApplier) Apply(c *types.Conf) (applied bool, err error) {
	cfg := buildConfigdb(c, a.interfaces)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}

	err = os.WriteFile(metalCoreConfigdbTmp, data, 0644)
	if err != nil {
		return false, err
	}

	applied, err = move(metalCoreConfigdbTmp, metalCoreConfigdb)
	if err != nil {
		return false, err
	}

	if applied {
		u := fmt.Sprintf("%s@%s.service", configdbReloadService, unit.UnitNamePathEscape(metalCoreConfigdb))
		return applied, dbus.Start(u)
	}

	return false, nil
}
