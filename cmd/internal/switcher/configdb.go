package switcher

import (
	"strconv"

	"github.com/go-redis/redis/v8"
)

const (
	loopbackTable    = "LOOPBACK_INTERFACE"
	vxlanTunnelTable = "VXLAN_TUNNEL"
)

type ConfigDBApplier struct {
	db *ConfigDB
}

func NewConfigDBApplier(cfg *SonicDatabaseConfig) *ConfigDBApplier {
	return &ConfigDBApplier{NewConfigDB(cfg)}
}

type VrfConfig struct {
	vlanIds    []string
	vrfs       []*vrf
	vlanIfaces []*vlanIface
	tunnelMaps []*vxlanTunnelMap
}

func build(cfg *Conf) *VrfConfig {
	c := &VrfConfig{
		vlanIds:    make([]string, 0, len(cfg.Ports.Vrfs)+1),
		vrfs:       make([]*vrf, 0, len(cfg.Ports.Vrfs)),
		vlanIfaces: make([]*vlanIface, 0, len(cfg.Ports.Vrfs)),
		tunnelMaps: make([]*vxlanTunnelMap, 0, len(cfg.Ports.Vrfs)),
	}

	for vrfName, v := range cfg.Ports.Vrfs {
		vlanId := strconv.FormatUint(uint64(v.VLANID), 10)
		vlanName := "Vlan" + vlanId
		vni := strconv.FormatUint(uint64(v.VNI), 10)

		c.vlanIds = append(c.vlanIds, vlanId)
		c.vlanIfaces = append(c.vlanIfaces, &vlanIface{name: vlanName, vrfName: vrfName})
		c.vrfs = append(c.vrfs, &vrf{name: vrfName, vni: vni})
		c.tunnelMaps = append(c.tunnelMaps, &vxlanTunnelMap{vlan: vlanName, vni: vni})
	}
	return c
}

func (a *ConfigDBApplier) Apply(cfg *Conf) error {
	c := build(cfg)

	// PXE Configuration
	c.vlanIds = append(c.vlanIds, "4000")
	c.vlanIfaces = append(c.vlanIfaces, &vlanIface{name: "Vlan4000", cidr: cfg.MetalCoreCIDR})
	c.tunnelMaps = append(c.tunnelMaps, &vxlanTunnelMap{vlan: "Vlan4000", vni: "104000"})

	err := configureVxlan(a.db, cfg.Loopback)
	if err != nil {
		return err
	}
	err = applyLoopback(a.db, cfg.Loopback)
	if err != nil {
		return err
	}

	err = applyVrfs(a.db, c.vrfs)
	if err != nil {
		return err
	}
	err = applyVlan(a.db, c.vlanIds)
	if err != nil {
		return err
	}
	err = applyVlanInterfaces(a.db, c.vlanIfaces)
	if err != nil {
		return err
	}
	err = applyVxlanTunnelMap(a.db, c.tunnelMaps)
	if err != nil {
		return err
	}

	err = applyVlanMember(a.db, "Vlan4000", cfg.Ports.Unprovisioned)
	if err != nil {
		return err
	}

	p := getPorts(cfg)
	err = applyMtus(a.db, p)
	if err != nil {
		return err
	}
	err = applyPorts(a.db, p)
	if err != nil {
		return err
	}
	return nil
}

func configureVxlan(db *ConfigDB, ip string) error {
	key := []string{vxlanTunnelTable, "vtep"}
	entry, err := db.GetEntry(key)
	if err == redis.Nil {
		return db.SetEntry(key, "src_ip", ip)
	}
	if err != nil {
		return err
	}
	if entry["src_ip"] != ip {
		return db.SetEntry(key, "src_ip", ip)
	}
	return nil
}

func applyLoopback(db *ConfigDB, ip string) error {
	view, err := db.GetView(loopbackTable)
	if err != nil {
		return err
	}

	infKey := []string{loopbackTable, "Loopback0"}
	if !view.Contains(infKey) {
		err = db.SetEntry(infKey)
		if err != nil {
			return err
		}
	}
	ipKey := []string{loopbackTable, "Loopback0", ip + "/32"}
	if !view.Contains(ipKey) {
		err = db.SetEntry(ipKey)
		if err != nil {
			return err
		}
	}

	view.Mask(infKey)
	view.Mask(ipKey)
	return view.DeleteUnmasked()
}
