package db

import (
	"context"

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

// ExistVlanMember reports whether vlanmgrd has added the interface to any vlan.
func (d *StateDB) ExistVlanMember(ctx context.Context, interfaceName string) (bool, error) {
	pattern := Key{"VLAN_MEMBER_TABLE", "*", interfaceName}
	keys, err := d.c.Keys(ctx, pattern)
	if err != nil {
		return false, err
	}

	return len(keys) > 0, nil
}
