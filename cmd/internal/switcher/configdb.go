package switcher

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/coreos/go-systemd/v22/unit"
)

const (
	metalCoreConfigdb     = "/etc/sonic/metal-core.json"
	configdbReloadService = "write-to-db"

	untagged = "untagged"
)

type configdb struct {
	Ifaces         map[string]*iface          `json:"INTERFACE"`
	Ports          map[string]*port           `json:"PORT"`
	Vlans          map[string]*vlan2          `json:"VLAN"`
	VlanIfaces     map[string]*iface          `json:"VLAN_INTERFACE"`
	VlanMembers    map[string]*vlanMember     `json:"VLAN_MEMBER"`
	Vrfs           map[string]*vrf            `json:"VRF"`
	VxlanTunnel    vxlanTunnel                `json:"VXLAN_TUNNEL"`
	VxlanTunnelMap map[string]*vxlanTunnelMap `json:"VXLAN_TUNNEL_MAP"`
}

type iface struct {
	VrfName string `json:"vrf_name,omitempty"`
}

type port struct {
	Mtu string `json:"mtu,omitempty"`
}

type vlan2 struct {
	VlanId string `json:"vlanid,omitempty"`
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

func buildConfigdb(cfg *Conf) *configdb {
	c := &configdb{
		Ifaces:         map[string]*iface{},
		Ports:          map[string]*port{},
		Vlans:          map[string]*vlan2{},
		VlanIfaces:     map[string]*iface{},
		VlanMembers:    map[string]*vlanMember{},
		Vrfs:           map[string]*vrf{},
		VxlanTunnel:    vxlanTunnel{vtep{SrcIp: cfg.Loopback}},
		VxlanTunnelMap: map[string]*vxlanTunnelMap{},
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
		c.Vlans[vlanName] = &vlan2{vlanId}
		c.VlanIfaces[vlanName] = &iface{vrfName}
		c.Vrfs[vrfName] = &vrf{vni}

		tunnelMapName := fmt.Sprintf("vtep|map_%s_%s", vni, vlanName)
		c.VxlanTunnelMap[tunnelMapName] = &vxlanTunnelMap{vlanName, vni}
	}
	pxeIfaceName := "Vlan4000|" + cfg.MetalCoreCIDR
	c.Vlans["Vlan4000"] = &vlan2{"4000"}
	c.VlanIfaces["Vlan4000"] = &iface{}
	c.VlanIfaces[pxeIfaceName] = &iface{}
	c.VxlanTunnelMap["vtep|map_104000_Vlan4000"] = &vxlanTunnelMap{"Vlan4000", "104000"}
	for _, p := range cfg.Ports.Unprovisioned {
		memberName := "Vlan4000|" + p
		c.Ifaces[p] = &iface{}
		c.Ports[p] = &port{Mtu: "9000"}
		c.VlanMembers[memberName] = &vlanMember{untagged}
	}

	return c
}

type configdbRenderer struct{}

func (r *configdbRenderer) Render(w io.Writer, cfg *Conf) error {
	c := buildConfigdb(cfg)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

type noopValidator struct{}

func (v *noopValidator) Validate(path string) error {
	return nil
}

func newConfigdbApplier() *networkApplier {
	d := newDestConfig(metalCoreConfigdb, &configdbRenderer{})
	reloadService := fmt.Sprintf("%s@%s.service", configdbReloadService, unit.UnitNamePathEscape(metalCoreConfigdb))
	r := dbusStartReloader{reloadService}

	return &networkApplier{
		destConfigs: []*destConfig{d},
		reloader:    &r,
		validator:   &noopValidator{},
	}
}