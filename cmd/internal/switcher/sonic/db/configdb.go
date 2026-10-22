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
	adminStatusField     = "admin_status"
	alias                = "alias"
	enable               = "enable"
	interfaceTable       = "INTERFACE"
	linkLocalOnly        = "ipv6_use_link_local_only" // nolint:gosec
	mtu                  = "mtu"
	portTable            = "PORT"
	taggingMode          = "tagging_mode"
	untagged             = "untagged"
	vlanMemberTable      = "VLAN_MEMBER"
	vrfTable             = "VRF"
	vrfName              = "vrf_name"
	vxlanTunnelMapTable  = "VXLAN_TUNNEL_MAP"
	lldpTable            = "LLDP"
	lldpTxIntervalModern = "tx_interval"
	lldpTxInterval202211 = "hello_time"
	lldpPortIdKey        = "lldp_local_portid"
)

func newConfigDB(rdb valkey.Client, sep string) *ConfigDB {
	return &ConfigDB{
		c: NewClient(rdb, sep),
	}
}

// EnsureEntry creates the entry or updates it if any of its fields differ.
// Fields of an existing entry that are not part of e.Val are left untouched.
func (d *ConfigDB) EnsureEntry(ctx context.Context, e Entry) error {
	current, err := d.c.HGetAll(ctx, e.Key)
	if err != nil {
		return err
	}

	val := e.Val
	if len(val) == 0 {
		if len(current) > 0 {
			return nil
		}
		// SONiC represents entries without fields with a NULL field
		val = Val{"NULL": "NULL"}
	}

	for field, value := range val {
		if v, ok := current[field]; !ok || v != value {
			return d.c.HSet(ctx, e.Key, val)
		}
	}
	return nil
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

type LldpSetTxIntervalMode int

const (
	Broadcom2022IntervalMode LldpSetTxIntervalMode = iota
	AsterNOSIntervalMode
)

func (d *ConfigDB) SetLldpTxInterval(ctx context.Context, mode LldpSetTxIntervalMode) error {
	switch mode {
	case Broadcom2022IntervalMode:
		return d.EnsureEntry(ctx, Entry{Key: Key{lldpTable, "GLOBAL"}, Val: Val{lldpTxInterval202211: "10"}})
	case AsterNOSIntervalMode:
		fallthrough
	default:
		return d.EnsureEntry(ctx, Entry{Key: Key{lldpTable, "lldp"}, Val: Val{lldpTxIntervalModern: "10"}})
	}
}

type LldpSetPortMode int

const (
	Broadcom2022PortMode LldpSetPortMode = iota
	AsterNOSPortMode
)

func (d *ConfigDB) SetLldpForPor(ctx context.Context, portName string, advertisedName string, mode LldpSetPortMode) error {
	switch mode {
	case Broadcom2022PortMode:
		// The lldpmgrd shipped in edgecore202211 is just peak
		// docker exec -it lldp cat /usr/bin/lldpmgrd if you want to take a look
		//
		// CONFIG_DB Example:
		//
		//	"LLDP_PORT|Ethernet1":	{
		//	  enabled: "true", # options: true and literally anything else
		//	  mode: "RECEIVE", # options: RECEIVE and literally anything else
		//	}
		//
		// enabled == "true" and mode unset is a KeyError in lldpmgrd
		// enabled == "true" means mode == "RECEIVE" means rx-only
		// enabled == "true" means mode != "RECEIVE" means tx-only (???)
		// LLDP_PORT|EthernetX KEY missing means tx-and-rx LLDP (???)
		// enabled != "true" means means disabled
		// enabled unset means "whatever the previous state was; Pray you never have to restart" (???)
		//
		// Are these people ok?
		_, err := d.DeleteEntry(ctx, Key{"LLDP_PORT", portName})
		return err
	case AsterNOSPortMode:
		fallthrough
	default:
		return d.EnsureEntry(ctx, Entry{Key: Key{lldpTable, portName}, Val: Val{lldpPortIdKey: advertisedName}})
	}

}
