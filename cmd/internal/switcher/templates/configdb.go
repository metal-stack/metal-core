package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/coreos/go-systemd/v22/unit"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const (
	// Tried to use "metal-core" name for the file. It doesn't work.
	// Systemd transforms "-" to "\" when %I specifier is used.
	metalCoreConfigdb     = "/etc/sonic/metal.json"
	metalCoreConfigdbTmp  = "/etc/sonic/metal.tmp"
	configdbReloadService = "write-to-db"
)

type configdb struct {
	Vlans          map[string]*vlan2          `json:"VLAN"`
	VlanIfaces     map[string]*iface          `json:"VLAN_INTERFACE"`
	Vrfs           map[string]*vrf            `json:"VRF"`
	VxlanTunnelMap map[string]*vxlanTunnelMap `json:"VXLAN_TUNNEL_MAP"`
}

type iface struct {
	VrfName string `json:"vrf_name,omitempty"`
}

type vlan2 struct {
	VlanId      string   `json:"vlanid,omitempty"`
	DHCPServers []string `json:"dhcp_servers,omitempty"`
}

type vrf struct {
	Vni string `json:"vni,omitempty"`
}

type vxlanTunnelMap struct {
	Vlan string `json:"vlan"`
	Vni  string `json:"vni"`
}

func buildConfigdb(cfg *types.Conf) *configdb {
	c := &configdb{
		Vlans:          map[string]*vlan2{},
		VlanIfaces:     map[string]*iface{},
		Vrfs:           map[string]*vrf{},
		VxlanTunnelMap: map[string]*vxlanTunnelMap{},
	}

	for vrfName, v := range cfg.Ports.Vrfs {
		vlanId := strconv.FormatUint(uint64(v.VLANID), 10)
		vlanName := "Vlan" + vlanId
		vni := strconv.FormatUint(uint64(v.VNI), 10)
		c.Vlans[vlanName] = &vlan2{VlanId: vlanId}
		c.VlanIfaces[vlanName] = &iface{vrfName}
		c.Vrfs[vrfName] = &vrf{vni}

		tunnelMapName := fmt.Sprintf("vtep|map_%s_%s", vni, vlanName)
		c.VxlanTunnelMap[tunnelMapName] = &vxlanTunnelMap{vlanName, vni}
	}

	return c
}

type ConfigdbApplier struct{}

func NewConfigdbApplier() *ConfigdbApplier {
	return &ConfigdbApplier{}
}

func (a *ConfigdbApplier) Apply(c *types.Conf) (applied bool, err error) {
	cfg := buildConfigdb(c)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}

	err = os.WriteFile(metalCoreConfigdbTmp, data, 0600)
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
