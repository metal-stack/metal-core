package db

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/valkey-io/valkey-go"
)

type (
	ConfigDB struct {
		c *Client
	}

	Port struct {
		Name        string
		Alias       string
		AdminStatus string
		Mtu         string
		// Index is the physical port number as defined in the platform port config.
		Index string
		// Lanes are the comma separated asic lanes of the port as defined in the platform port config.
		Lanes string
	}

	VxlanMap struct {
		Vni  string
		Vlan string
	}
)

const (
	aclTable            = "ACL_TABLE"
	aclTablePorts       = "ports@"
	adminStatusField    = "admin_status"
	alias               = "alias"
	enable              = "enable"
	index               = "index"
	interfaceTable      = "INTERFACE"
	lanes               = "lanes"
	null                = "NULL"
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
	vni                 = "vni"
	vrfName             = "vrf_name"
	vxlanTunnelMapTable = "VXLAN_TUNNEL_MAP"
)

func newConfigDB(rdb valkey.Client, sep string) *ConfigDB {
	return &ConfigDB{
		c: NewClient(rdb, sep),
	}
}

// Client returns the underlying redis client, intended for tests.
func (d *ConfigDB) Client() *Client {
	return d.c
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

// GetVrfVni returns the vni of the given vrf. If the vrf has no vni field, the vni is derived from the vrf name (Vrf<vni>).
func (d *ConfigDB) GetVrfVni(ctx context.Context, vrf string) (uint32, error) {
	key := Key{vrfTable, vrf}

	value, err := d.c.HGet(ctx, key, vni)
	if err != nil {
		return 0, err
	}
	if value == "" {
		value = strings.TrimPrefix(vrf, "Vrf")
	}

	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("could not determine vni of vrf %s: %w", vrf, err)
	}
	return uint32(parsed), nil // nolint:gosec
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

// ExistInterfaceConfiguration returns true if the interface has an entry in the INTERFACE table, i.e. is a routed interface.
func (d *ConfigDB) ExistInterfaceConfiguration(ctx context.Context, interfaceName string) (bool, error) {
	key := Key{interfaceTable, interfaceName}

	return d.c.Exists(ctx, key)
}

// DeleteInterfaceConfiguration removes the interface from the INTERFACE table including all of its addresses.
func (d *ConfigDB) DeleteInterfaceConfiguration(ctx context.Context, interfaceName string) error {
	addresses, err := d.GetInterfaceAddresses(ctx, interfaceName)
	if err != nil {
		return err
	}
	for _, address := range addresses {
		if err := d.DeleteInterfaceAddress(ctx, interfaceName, address); err != nil {
			return err
		}
	}

	key := Key{interfaceTable, interfaceName}

	return d.c.Del(ctx, key)
}

// GetInterfaceAddresses returns the addresses (in cidr notation) configured on the interface.
func (d *ConfigDB) GetInterfaceAddresses(ctx context.Context, interfaceName string) ([]string, error) {
	pattern := Key{interfaceTable, interfaceName, "*"}

	keys, err := d.c.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	addresses := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key) != 3 {
			return nil, fmt.Errorf("could not parse key %v", key)
		}
		addresses = append(addresses, key[2])
	}
	slices.Sort(addresses)
	return addresses, nil
}

// SetInterfaceAddress adds the address (in cidr notation) to the interface. The interface must already exist in the INTERFACE table.
func (d *ConfigDB) SetInterfaceAddress(ctx context.Context, interfaceName, address string) error {
	key := Key{interfaceTable, interfaceName, address}

	// an entry without fields can not be stored in redis, sonic uses a NULL field for this purpose
	return d.c.HSet(ctx, key, Val{null: null})
}

func (d *ConfigDB) DeleteInterfaceAddress(ctx context.Context, interfaceName, address string) error {
	key := Key{interfaceTable, interfaceName, address}

	return d.c.Del(ctx, key)
}

// GetACLTables returns the names of all acl tables.
func (d *ConfigDB) GetACLTables(ctx context.Context) ([]string, error) {
	t := d.c.GetTable(Key{aclTable})

	res, err := t.GetView(ctx)
	if err != nil {
		return nil, err
	}

	tables := make([]string, 0, len(res))
	for name := range res {
		tables = append(tables, name)
	}
	slices.Sort(tables)
	return tables, nil
}

// GetACLTablePorts returns the ports the acl table is bound to.
func (d *ConfigDB) GetACLTablePorts(ctx context.Context, table string) ([]string, error) {
	key := Key{aclTable, table}

	value, err := d.c.HGet(ctx, key, aclTablePorts)
	if err != nil {
		return nil, err
	}

	ports := make([]string, 0)
	for _, port := range strings.Split(value, ",") {
		port = strings.TrimSpace(port)
		if port == "" {
			continue
		}
		ports = append(ports, port)
	}
	slices.Sort(ports)
	return ports, nil
}

// SetACLTablePorts binds the acl table to exactly the given ports, an empty list unbinds the table from all ports.
func (d *ConfigDB) SetACLTablePorts(ctx context.Context, table string, ports []string) error {
	key := Key{aclTable, table}

	ports = slices.Clone(ports)
	slices.Sort(ports)
	ports = slices.Compact(ports)
	return d.c.HSet(ctx, key, Val{aclTablePorts: strings.Join(ports, ",")})
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

	if len(result) == 0 {
		return nil, nil
	}

	return &Port{
		Name:        interfaceName,
		Alias:       result[alias],
		AdminStatus: result[adminStatusField],
		Mtu:         result[mtu],
		Index:       result[index],
		Lanes:       result[lanes],
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
			AdminStatus: result[adminStatusField],
			Mtu:         result[mtu],
			Index:       result[index],
			Lanes:       result[lanes],
		})
	}

	return ports, nil
}

func (d *ConfigDB) SetPortMtu(ctx context.Context, interfaceName string, val string) error {
	key := Key{portTable, interfaceName}

	return d.c.HSet(ctx, key, Val{mtu: val})
}

func (d *ConfigDB) SetAdminStatus(ctx context.Context, interfaceName string, adminStatus types.PortStatus) error {
	if adminStatus != types.PortStatusDown && adminStatus != types.PortStatusUp {
		return fmt.Errorf("unknown admin status %s", adminStatus)
	}
	key := Key{portTable, interfaceName}
	return d.c.HSet(ctx, key, Val{adminStatusField: string(adminStatus)})
}
