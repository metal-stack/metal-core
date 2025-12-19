package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	adminStatus         = "admin_status"
	adminStatusUp       = "up"
	adminStatusDown     = "down"
	alias               = "alias"
	enable              = "enable"
	interfaceTable      = "INTERFACE"
	linkLocalOnly       = "ipv6_use_link_local_only" // nolint:gosec
	mtu                 = "mtu"
	portTable           = "PORT"
	suppressVlanNeigh   = "SUPPRESS_VLAN_NEIGH"
	taggingMode         = "tagging_mode"
	untagged            = "untagged"
	vlanTable           = "VLAN"
	vlanInterfaceTable  = "VLAN_INTERFACE"
	vlanMemberTable     = "VLAN_MEMBER"
	vrfTable            = "VRF"
	vrfName             = "vrf_name"
	vxlanTunnelMapTable = "VXLAN_TUNNEL_MAP"
)

type ConfigDB struct {
	c *Client
}

type Port struct {
	Name        string
	Alias       string
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
	key := Key{vlanTable, fmt.Sprintf("Vlan%d", vid)}

	return d.c.Exists(ctx, key)
}

func (d *ConfigDB) CreateVlan(ctx context.Context, vid uint16) error {
	vlanId := fmt.Sprintf("%d", vid)
	key := Key{vlanTable, "Vlan" + vlanId}

	return d.c.HSet(ctx, key, Val{"vlanid": vlanId})
}

func (d *ConfigDB) DeleteVlan(ctx context.Context, vid uint16) error {
	vlanId := fmt.Sprintf("%d", vid)
	key := Key{vlanTable, "Vlan" + vlanId}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) AreNeighborsSuppressed(ctx context.Context, vid uint16) (bool, error) {
	key := Key{suppressVlanNeigh, fmt.Sprintf("Vlan%d", vid)}

	suppress, err := d.c.HGet(ctx, key, "suppress")
	if err != nil {
		return false, err
	}
	return suppress == "on", nil
}

func (d *ConfigDB) SuppressNeighbors(ctx context.Context, vid uint16) error {
	key := Key{suppressVlanNeigh, fmt.Sprintf("Vlan%d", vid)}

	return d.c.HSet(ctx, key, Val{"suppress": "on"})
}

func (d *ConfigDB) DeleteNeighborSuppression(ctx context.Context, vid uint16) error {
	key := Key{suppressVlanNeigh, fmt.Sprintf("Vlan%d", vid)}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) ExistVlanInterface(ctx context.Context, vid uint16) (bool, error) {
	key := Key{vlanInterfaceTable, fmt.Sprintf("Vlan%d", vid)}

	return d.c.Exists(ctx, key)
}

func (d *ConfigDB) CreateVlanInterface(ctx context.Context, vid uint16, vrf string) error {
	key := Key{vlanInterfaceTable, "Vlan" + fmt.Sprintf("%d", vid)}

	return d.c.HSet(ctx, key, Val{vrfName: vrf})
}

func (d *ConfigDB) DeleteVlanInterface(ctx context.Context, vid uint16) error {
	key := Key{vlanInterfaceTable, "Vlan" + fmt.Sprintf("%d", vid)}

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
	t := d.c.GetTable(Key{vrfTable})

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
	key := Key{vrfTable, vrf}

	return d.c.Exists(ctx, key)
}

func (d *ConfigDB) CreateVrf(ctx context.Context, vrf string, vni uint32) error {
	key := Key{vrfTable, vrf}

	return d.c.HSet(ctx, key, Val{"fallback": "false", "vni": fmt.Sprintf("%d", vni)})
}

func (d *ConfigDB) DeleteVrf(ctx context.Context, vrf string) error {
	key := Key{vrfTable, vrf}

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
	vtep, err := d.getVTEPName(ctx)
	if err != nil {
		return false, err
	}
	key := Key{vxlanTunnelMapTable, vtep, fmt.Sprintf("map_%d_Vlan%d", vni, vid)}

	return d.c.Exists(ctx, key)
}

func (d *ConfigDB) CreateVxlanTunnelMap(ctx context.Context, vid uint16, vni uint32) error {
	vtep, err := d.getVTEPName(ctx)
	if err != nil {
		return err
	}
	key := Key{vxlanTunnelMapTable, vtep, fmt.Sprintf("map_%d_Vlan%d", vni, vid)}
	val := Val{
		"vlan": fmt.Sprintf("Vlan%d", vid),
		"vni":  fmt.Sprintf("%d", vni),
	}
	return d.c.HSet(ctx, key, val)
}

func (d *ConfigDB) DeleteVxlanTunnelMap(ctx context.Context, vid uint16, vni uint32) error {
	vtep, err := d.getVTEPName(ctx)
	if err != nil {
		return err
	}
	key := Key{vxlanTunnelMapTable, vtep, fmt.Sprintf("map_%d_Vlan%d", vni, vid)}

	return d.c.Del(ctx, key)
}

func (d *ConfigDB) FindVxlanTunnelMapByVni(ctx context.Context, vni uint32) (*VxlanMap, error) {
	vtep, err := d.getVTEPName(ctx)
	if err != nil {
		return nil, err
	}
	t := d.c.GetTable(Key{vxlanTunnelMapTable, vtep})

	res, err := t.GetView(ctx)
	if err != nil {
		return nil, err
	}

	tunnelMaps := make([]string, 0)
	for k := range res {
		tunnelMaps = append(tunnelMaps, k)
	}

	for _, k := range tunnelMaps {
		result, err := d.c.HGetAll(ctx, Key{vxlanTunnelMapTable, vtep, k})
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

func (d *ConfigDB) getVTEPName(ctx context.Context) (string, error) {
	pattern := Key{"VXLAN_TUNNEL", "*"}
	keys, err := d.c.Keys(ctx, pattern)
	if err != nil {
		return "", err
	}
	if len(keys) != 1 {
		return "", fmt.Errorf("could not find name of the vtep")
	}
	key := []string(keys[0])
	return key[len(key)-1], nil
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

func (d *ConfigDB) GetPorts(ctx context.Context) ([]*Port, error) {
	var (
		ports     []*Port
		portNames []string
	)

	t := d.c.GetTable(Key{portTable})
	res, err := t.GetView(ctx)
	if err != nil {
		return nil, err
	}

	for name := range res {
		portNames = append(portNames, name)
	}

	for _, p := range portNames {
		result, err := d.c.HGetAll(ctx, Key{portTable, p})
		if err != nil {
			return nil, err
		}

		ports = append(ports, &Port{
			Name:        p,
			Alias:       result[alias],
			AdminStatus: result[adminStatus] == adminStatusUp,
			Mtu:         result[mtu],
		})
	}

	return ports, nil
}

func (d *ConfigDB) SetPortMtu(ctx context.Context, interfaceName string, val string) error {
	key := Key{portTable, interfaceName}

	return d.c.HSet(ctx, key, Val{mtu: val})
}

func (d *ConfigDB) SetAdminStatusUp(ctx context.Context, interfaceName string, up bool) error {
	key := Key{portTable, interfaceName}

	status := adminStatusUp
	if !up {
		status = adminStatusDown
	}
	return d.c.HSet(ctx, key, Val{adminStatus: status})
}
