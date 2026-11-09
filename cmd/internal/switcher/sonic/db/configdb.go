package db

import (
	"context"
	"fmt"

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
	}

	VxlanMap struct {
		Vni  string
		Vlan string
	}

	// Entry is a CONFIG_DB entry with the fields it should have.
	Entry struct {
		Key Key
		Val Val
	}
)

const (
	adminStatusField    = "admin_status"
	alias               = "alias"
	enable              = "enable"
	interfaceTable      = "INTERFACE"
	linkLocalOnly       = "ipv6_use_link_local_only" // nolint:gosec
	mtu                 = "mtu"
	portTable           = "PORT"
	taggingMode         = "tagging_mode"
	untagged            = "untagged"
	vlanMemberTable     = "VLAN_MEMBER"
	vrfTable            = "VRF"
	vrfName             = "vrf_name"
	vxlanTunnelMapTable = "VXLAN_TUNNEL_MAP"
	null                = "NULL"
)

func newConfigDB(rdb valkey.Client, sep string) *ConfigDB {
	return &ConfigDB{
		c: NewClient(rdb, sep),
	}
}

// MergeEntry adds or updates the fields in e.Val while leaving all other fields untouched.
// An empty value is a no-op.
func (d *ConfigDB) MergeEntry(ctx context.Context, e Entry) error {
	if len(e.Val) == 0 {
		return nil
	}

	current, err := d.c.HGetAll(ctx, e.Key)
	if err != nil {
		return err
	}

	for field, value := range e.Val {
		if v, ok := current[field]; !ok || v != value {
			if err := d.c.HSet(ctx, e.Key, e.Val); err != nil {
				return err
			}
			break
		}
	}

	// NULL is an internal placeholder for entries without fields. It must not
	// remain once the entry has real fields.
	if _, hasNull := current[null]; hasNull {
		if _, wantsNull := e.Val[null]; !wantsNull {
			return d.c.HDel(ctx, e.Key, null)
		}
	}
	return nil
}

// EnsureExactEntry creates the entry or makes its fields exactly match e.Val.
// SONiC represents entries without fields with a NULL field.
func (d *ConfigDB) EnsureExactEntry(ctx context.Context, e Entry) error {
	current, err := d.c.HGetAll(ctx, e.Key)
	if err != nil {
		return err
	}

	desired := e.Val
	if len(desired) == 0 {
		desired = Val{null: null}
	}

	for field, value := range desired {
		if v, ok := current[field]; !ok || v != value {
			if err := d.c.HSet(ctx, e.Key, desired); err != nil {
				return err
			}
			break
		}
	}

	var stale []string
	for field := range current {
		if _, ok := desired[field]; !ok {
			stale = append(stale, field)
		}
	}
	return d.c.HDel(ctx, e.Key, stale...)
}

// DeleteEntry deletes the entry and reports whether it existed.
func (d *ConfigDB) DeleteEntry(ctx context.Context, key Key) (bool, error) {
	exists, err := d.c.Exists(ctx, key)
	if err != nil || !exists {
		return false, err
	}
	return true, d.c.Del(ctx, key)
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

func (d *ConfigDB) SetVrfMember(ctx context.Context, interfaceName string, vrf string) error {
	key := Key{interfaceTable, interfaceName}

	return d.c.HSet(ctx, key, Val{linkLocalOnly: enable, vrfName: vrf})
}

func (d *ConfigDB) GetVrfMembership(ctx context.Context, interfaceName string) (string, error) {
	key := Key{interfaceTable, interfaceName}

	return d.c.HGet(ctx, key, vrfName)
}

func (d *ConfigDB) FindVxlanTunnelMapByVni(ctx context.Context, vni uint32) (*VxlanMap, error) {
	vtep, err := d.GetVTEPName(ctx)
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

func (d *ConfigDB) GetVTEPName(ctx context.Context) (string, error) {
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

	if len(result) == 0 {
		return nil, nil
	}

	return &Port{
		Name:        interfaceName,
		Alias:       result[alias],
		AdminStatus: result[adminStatusField],
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
			AdminStatus: result[adminStatusField],
			Mtu:         result[mtu],
		})
	}

	return ports, nil
}

func (d *ConfigDB) SetPortMtu(ctx context.Context, interfaceName string, val string) error {
	key := Key{portTable, interfaceName}

	return d.c.HSet(ctx, key, Val{mtu: val})
}

func (d *ConfigDB) GetAdminStatus(ctx context.Context, interfaceName string) (types.PortStatus, error) {
	key := Key{portTable, interfaceName}

	status, err := d.c.HGet(ctx, key, adminStatusField)
	if err != nil {
		return "", err
	}

	return types.PortStatus(status), nil
}

func (d *ConfigDB) SetAdminStatus(ctx context.Context, interfaceName string, adminStatus types.PortStatus) error {
	if adminStatus != types.PortStatusDown && adminStatus != types.PortStatusUp {
		return fmt.Errorf("unknown admin status %s", adminStatus)
	}
	key := Key{portTable, interfaceName}
	return d.c.HSet(ctx, key, Val{adminStatusField: string(adminStatus)})
}
