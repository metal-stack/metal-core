package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
)

const (
	// Tried to use name that includes "-" for the file. It doesn't work.
	// Systemd transforms "-" to "\" when %I specifier is used.
	metalCoreVrfConfig    = "/etc/sonic/metal/vrfs.json"
	metalCoreVrfConfigTmp = "/etc/sonic/metal/vrfs.tmp"

	metalCoreVlanConfig    = "/etc/sonic/metal/vlans.json"
	metalCoreVlanConfigTmp = "/etc/sonic/metal/vlans.tmp"

	configdbUpdateService = "write-to-db"

	untagged = "untagged"
)

type vrfConfig struct {
	Ifaces         map[string]*iface          `json:"INTERFACE"`
	Vlans          map[string]*vlan           `json:"VLAN"`
	VlanIfaces     map[string]*iface          `json:"VLAN_INTERFACE"`
	Vrfs           map[string]*vrf            `json:"VRF"`
	VxlanTunnelMap map[string]*vxlanTunnelMap `json:"VXLAN_TUNNEL_MAP"`
}

type vlanConfig struct {
	VlanMembers map[string]*vlanMember `json:"VLAN_MEMBER"`
}

type iface struct {
	VrfName string `json:"vrf_name,omitempty"`
}

type vlan struct {
	VlanId string `json:"vlanid,omitempty"`
}

type vlanMember struct {
	TaggingMode string `json:"tagging_mode"`
}

type vrf struct {
	Vni string `json:"vni,omitempty"`
}

type vxlanTunnelMap struct {
	Vlan string `json:"vlan"`
	Vni  string `json:"vni"`
}

type ConfigdbApplier struct {
	interfaces []string
}

func NewConfigdbApplier(infs []string) *ConfigdbApplier {
	return &ConfigdbApplier{
		interfaces: infs,
	}
}

func (a *ConfigdbApplier) Apply(cfg *types.Conf) (applied bool, err error) {
	vrfApplied, err := a.applyVrfConfig(cfg)
	if err != nil {
		return false, fmt.Errorf("failed to apply VRF config: %w", err)
	}

	vlanApplied, err := a.applyVlanConfig(cfg)
	if err != nil {
		return false, fmt.Errorf("failed to apply VLAN config: %w", err)
	}

	return vrfApplied || vlanApplied, nil
}

func buildVrfConfig(cfg *types.Conf, fpInfs []string) *vrfConfig {
	c := &vrfConfig{
		Vrfs:           map[string]*vrf{},
		Ifaces:         map[string]*iface{},
		Vlans:          map[string]*vlan{},
		VlanIfaces:     map[string]*iface{},
		VxlanTunnelMap: map[string]*vxlanTunnelMap{},
	}

	for vrfName, v := range cfg.Ports.Vrfs {
		for _, p := range v.Neighbors {
			c.Ifaces[p] = &iface{vrfName}
		}
		vlanId := strconv.FormatUint(uint64(v.VLANID), 10)
		vlanName := "Vlan" + vlanId
		vni := strconv.FormatUint(uint64(v.VNI), 10)
		c.Vlans[vlanName] = &vlan{VlanId: vlanId}
		c.VlanIfaces[vlanName] = &iface{vrfName}
		c.Vrfs[vrfName] = &vrf{vni}

		tunnelMapName := fmt.Sprintf("vtep|map_%s_%s", vni, vlanName)
		c.VxlanTunnelMap[tunnelMapName] = &vxlanTunnelMap{vlanName, vni}
	}

	// Configure interfaces
	for _, p := range cfg.Ports.Underlay {
		c.Ifaces[p] = &iface{}
	}
	for _, fw := range cfg.Ports.Firewalls {
		c.Ifaces[fw.Port] = &iface{}
	}
	for _, p := range cfg.Ports.Unprovisioned {
		c.Ifaces[p] = &iface{}
	}

	// Remove IPs for front-panel interfaces
	for _, inf := range fpInfs {
		if strings.Contains(inf, "|") {
			c.Ifaces[inf] = nil
		}
	}

	return c
}

func (a *ConfigdbApplier) applyVrfConfig(cfg *types.Conf) (applied bool, err error) {
	vrfConfig := buildVrfConfig(cfg, a.interfaces)

	data, err := json.MarshalIndent(vrfConfig, "", "  ")
	if err != nil {
		return false, err
	}

	err = os.WriteFile(metalCoreVrfConfigTmp, data, 0600)
	if err != nil {
		return false, err
	}

	applied, err = move(metalCoreVrfConfigTmp, metalCoreVrfConfig)
	if err != nil {
		return false, err
	}

	if applied {
		u := fmt.Sprintf("%s@%s.service", configdbUpdateService, unit.UnitNamePathEscape(metalCoreVrfConfig))
		return applied, dbus.Start(u)
	}

	return false, nil
}

func buildVlanConfig(cfg *types.Conf) *vlanConfig {
	c := &vlanConfig{
		VlanMembers: map[string]*vlanMember{},
	}

	for _, p := range cfg.Ports.Provisioned {
		memberName := "Vlan4000|" + p
		c.VlanMembers[memberName] = nil
	}
	for _, p := range cfg.Ports.Unprovisioned {
		memberName := "Vlan4000|" + p
		c.VlanMembers[memberName] = &vlanMember{untagged}
	}

	return c
}

func (a *ConfigdbApplier) applyVlanConfig(cfg *types.Conf) (applied bool, err error) {
	vlanConfig := buildVlanConfig(cfg)

	data, err := json.MarshalIndent(vlanConfig, "", "  ")
	if err != nil {
		return false, err
	}

	err = os.WriteFile(metalCoreVlanConfigTmp, data, 0600)
	if err != nil {
		return false, err
	}

	applied, err = move(metalCoreVlanConfigTmp, metalCoreVlanConfig)
	if err != nil {
		return false, err
	}

	if applied {
		u := fmt.Sprintf("%s@%s.service", configdbUpdateService, unit.UnitNamePathEscape(metalCoreVlanConfig))
		return applied, dbus.Start(u)
	}

	return false, nil
}
