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
