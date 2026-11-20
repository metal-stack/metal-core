package db

import (
	"context"
	"maps"
	"slices"

	"github.com/valkey-io/valkey-go"
)

type StateDB struct {
	c *Client
}

func newStateDB(rdb valkey.Client, sep string) *StateDB {
	return &StateDB{
		c: NewClient(rdb, sep),
	}
}

// ExistInterface reports whether intfmgrd has set up the interface as router interface.
func (d *StateDB) ExistInterface(ctx context.Context, interfaceName string) (bool, error) {
	key := Key{"INTERFACE_TABLE", interfaceName}

	return d.c.Exists(ctx, key)
}

// GetInterfaceVrf returns the vrf intfmgrd has bound the interface to, and whether it has set up the interface at all.
func (d *StateDB) GetInterfaceVrf(ctx context.Context, interfaceName string) (vrf string, exists bool, err error) {
	val, err := d.c.HGetAll(ctx, Key{"INTERFACE_TABLE", interfaceName})
	if err != nil {
		return "", false, err
	}
	return val["vrf"], len(val) > 0, nil
}

// GetVlans returns the vlans vlanmgrd has set up.
func (d *StateDB) GetVlans(ctx context.Context) ([]string, error) {
	view, err := d.c.GetView(ctx, "VLAN_TABLE")
	if err != nil {
		return nil, err
	}

	return slices.Collect(maps.Keys(view)), nil
}

// ExistVlanMember reports whether vlanmgrd has added the interface to any of the given vlans.
func (d *StateDB) ExistVlanMember(ctx context.Context, interfaceName string, vlans []string) (bool, error) {
	for _, vlan := range vlans {
		exists, err := d.c.Exists(ctx, Key{"VLAN_MEMBER_TABLE", vlan, interfaceName})
		if err != nil || exists {
			return exists, err
		}
	}

	return false, nil
}
