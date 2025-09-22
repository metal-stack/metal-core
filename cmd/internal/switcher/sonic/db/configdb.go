package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	enable          = "enable"
	interfaceTable  = "INTERFACE"
	linkLocalOnly   = "ipv6_use_link_local_only" // nolint:gosec
	vlanMemberTable = "VLAN_MEMBER"
	taggingMode     = "tagging_mode"
	untagged        = "untagged"
	vrfName         = "vrf_name"
	portTable       = "PORT"
	adminStatus     = "admin_status"
	adminStatusUp   = "up"
	mtu             = "mtu"
)

type ConfigDB struct {
	c    *Client
	vtep string
}

type Port struct {
	AdminStatus bool
	Mtu         string
}

type VxlanMap struct {
	Vni  string
	Vlan string
}

func newConfigDB(rdb *redis.Client, sep string) *ConfigDB {
	return &ConfigDB{
		c: NewClient(rdb, sep),
	}
}

func (d *ConfigDB) ExistVlan(ctx context.Context, vid uint16) (bool, error) {
	key := Key{"VLAN", fmt.Sprintf("Vlan%d", vid)}

	return d.c.Exists(ctx, key)
}

func (d *ConfigDB) CreateVlan(ctx context.Context, vid uint16) error {
	vlanId := fmt.Sprintf("%d", vid)
	key := Key{"VLAN", "Vlan" + vlanId}

	return d.c.HSet(ctx, key, Val{"admin_status": "up", "vlanid": vlanId})
}

func (d *ConfigDB) DeleteVlan(ctx context.Context, vid uint16) error {
	vlanId := fmt.Sprintf("%d", vid)
	key := Key{"VLAN", "Vlan" + vlanId}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) AreNeighborsSuppressed(ctx context.Context, vid uint16) (bool, error) {
	key := Key{"SUPPRESS_VLAN_NEIGH", fmt.Sprintf("Vlan%d", vid)}

	suppress, err := d.c.HGet(ctx, key, "suppress")
	if err != nil {
		return false, err
	}
	return suppress == "on", nil
}

func (d *ConfigDB) SuppressNeighbors(ctx context.Context, vid uint16) error {
	key := Key{"SUPPRESS_VLAN_NEIGH", fmt.Sprintf("Vlan%d", vid)}

	return d.c.HSet(ctx, key, Val{"suppress": "on"})
}

func (d *ConfigDB) DeleteNeighborSuppression(ctx context.Context, vid uint16) error {
	key := Key{"SUPPRESS_VLAN_NEIGH", fmt.Sprintf("Vlan%d", vid)}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) ExistVlanInterface(ctx context.Context, vid uint16) (bool, error) {
	key := Key{"VLAN_INTERFACE", fmt.Sprintf("Vlan%d", vid)}

	return d.c.Exists(ctx, key)
}

func (d *ConfigDB) CreateVlanInterface(ctx context.Context, vid uint16, vrf string) error {
	key := Key{"VLAN_INTERFACE", "Vlan" + fmt.Sprintf("%d", vid)}

	return d.c.HSet(ctx, key, Val{vrfName: vrf})
}

func (d *ConfigDB) DeleteVlanInterface(ctx context.Context, vid uint16) error {
	key := Key{"VLAN_INTERFACE", "Vlan" + fmt.Sprintf("%d", vid)}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) GetVlanMembership(ctx context.Context, interfaceName string) ([]string, error) {
	pattern := Key{vlanMemberTable, "*", interfaceName}

	keys, err := d.c.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	vlans := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key) != 3 {
			return nil, fmt.Errorf("could not parse key %v", key)
		}
		vlans = append(vlans, key[1])
	}
	return vlans, nil
}

func (d *ConfigDB) SetVlanMember(ctx context.Context, interfaceName, vlan string) error {
	key := Key{vlanMemberTable, vlan, interfaceName}

	return d.c.HSet(ctx, key, Val{taggingMode: untagged})
}

func (d *ConfigDB) DeleteVlanMember(ctx context.Context, interfaceName, vlan string) error {
	key := Key{vlanMemberTable, vlan, interfaceName}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) GetVrfs(ctx context.Context) ([]string, error) {
	t := d.c.GetTable(Key{"VRF"})

	res, err := t.GetView(ctx)
	if err != nil {
		return nil, err
	}

	vrfs := make([]string, 0)
	for vrf := range res {
		vrfs = append(vrfs, vrf)
	}

	return vrfs, nil
}

func (d *ConfigDB) ExistVrf(ctx context.Context, vrf string) (bool, error) {
	key := Key{"VRF", vrf}

	return d.c.Exists(ctx, key)
}

func (d *ConfigDB) CreateVrf(ctx context.Context, vrf string, vni uint32) error {
	key := Key{"VRF", vrf}

	return d.c.HSet(ctx, key, Val{"fallback": "false", "vni": fmt.Sprintf("%d", vni)})
}

func (d *ConfigDB) DeleteVrf(ctx context.Context, vrf string) error {
	key := Key{"VRF", vrf}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) SetVrfMember(ctx context.Context, interfaceName string, vrf string) error {
	key := Key{interfaceTable, interfaceName}

	return d.c.HSet(ctx, key, Val{linkLocalOnly: enable, vrfName: vrf})
}

func (d *ConfigDB) GetVrfMembership(ctx context.Context, interfaceName string) (string, error) {
	key := Key{interfaceTable, interfaceName}

	return d.c.HGet(ctx, key, vrfName)
}

func (d *ConfigDB) ExistVxlanTunnelMap(ctx context.Context, vid uint16, vni uint32) (bool, error) {
	if d.vtep == "" {
		if err := d.obtainVTEPName(ctx); err != nil {
			return false, err
		}
	}
	key := Key{"VXLAN_TUNNEL_MAP", d.vtep, fmt.Sprintf("map_%d_Vlan%d", vni, vid)}

	return d.c.Exists(ctx, key)
}

func (d *ConfigDB) CreateVxlanTunnelMap(ctx context.Context, vid uint16, vni uint32) error {
	if d.vtep == "" {
		if err := d.obtainVTEPName(ctx); err != nil {
			return err
		}
	}
	key := Key{"VXLAN_TUNNEL_MAP", d.vtep, fmt.Sprintf("map_%d_Vlan%d", vni, vid)}
	val := Val{
		"vlan": fmt.Sprintf("Vlan%d", vid),
		"vni":  fmt.Sprintf("%d", vni),
	}
	return d.c.HSet(ctx, key, val)
}

func (d *ConfigDB) obtainVTEPName(ctx context.Context) error {
	pattern := Key{"VXLAN_TUNNEL", "*"}
	keys, err := d.c.Keys(ctx, pattern)
	if err != nil {
		return err
	}
	if len(keys) != 1 {
		return fmt.Errorf("could not find name of the vtep")
	}
	key := []string(keys[0])
	d.vtep = key[len(key)-1]
	return nil
}

func (d *ConfigDB) DeleteVxlanTunnelMap(ctx context.Context, vid uint16, vni uint32) error {
	key := Key{"VXLAN_TUNNEL_MAP", "vtep", fmt.Sprintf("map_%d_Vlan%d", vni, vid)}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) FindVxlanTunnelMapByVni(ctx context.Context, vni uint32) (*VxlanMap, error) {
	t := d.c.GetTable(Key{"VXLAN_TUNNEL_MAP", "vtep"})

	res, err := t.GetView(ctx)
	if err != nil {
		return nil, err
	}

	tunnelMaps := make([]string, 0)
	for k := range res {
		tunnelMaps = append(tunnelMaps, k)
	}

	for _, k := range tunnelMaps {
		result, err := d.c.HGetAll(ctx, Key{"VXLAN_TUNNEL_MAP", "vtep", k})
		if err != nil {
			return nil, err
		}

		if result["vni"] == fmt.Sprintf("%d", vni) {
			return &VxlanMap{
				Vni:  result["vni"],
				Vlan: result["vlan"],
			}, nil
		}
	}

	return nil, nil
}

func (d *ConfigDB) DeleteInterfaceConfiguration(ctx context.Context, interfaceName string) error {
	key := Key{interfaceTable, interfaceName}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) IsLinkLocalOnly(ctx context.Context, interfaceName string) (bool, error) {
	key := Key{interfaceTable, interfaceName}

	result, err := d.c.HGet(ctx, key, linkLocalOnly)
	if err != nil {
		return false, err
	}
	return result == enable, nil
}

func (d *ConfigDB) EnableLinkLocalOnly(ctx context.Context, interfaceName string) error {
	key := Key{interfaceTable, interfaceName}

	return d.c.HSet(ctx, key, Val{linkLocalOnly: enable})
}

func (d *ConfigDB) GetPort(ctx context.Context, interfaceName string) (*Port, error) {
	key := Key{portTable, interfaceName}

	result, err := d.c.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}

	return &Port{
		AdminStatus: result[adminStatus] == adminStatusUp,
		Mtu:         result[mtu],
	}, nil
}

func (d *ConfigDB) SetPortMtu(ctx context.Context, interfaceName string, val string) error {
	key := Key{portTable, interfaceName}

	return d.c.HSet(ctx, key, Val{mtu: val})
}

func (d *ConfigDB) SetAdminStatusUp(ctx context.Context, interfaceName string) error {
	key := Key{portTable, interfaceName}

	return d.c.HSet(ctx, key, Val{adminStatus: adminStatusUp})
}
