package db

import (
	"context"
	"slices"
	"strings"

	"github.com/valkey-io/valkey-go"
)

type ApplDB struct {
	c *Client
}

func newApplDB(rdb valkey.Client, sep string) *ApplDB {
	return &ApplDB{
		c: NewClient(rdb, sep),
	}
}

func (d *ApplDB) ExistPortInitDone(ctx context.Context) (bool, error) {
	key := Key{"PORT_TABLE", "PortInitDone"}

	return d.c.Exists(ctx, key)
}

// GetRoutesVia returns the keys of the routes fpmsyncd installed in the vrf that use the interface as next hop.
// An empty vrf returns the routes of all vrfs.
func (d *ApplDB) GetRoutesVia(ctx context.Context, vrf, interfaceName string) ([]string, error) {
	pattern := Key{"ROUTE_TABLE", vrf, "*"}
	if vrf == "" {
		pattern = Key{"ROUTE_TABLE", "*"}
	}

	routes, err := d.c.HGetAllMatching(ctx, pattern)
	if err != nil {
		return nil, err
	}

	var keys []string
	for key, route := range routes {
		// next hops of ECMP routes are comma separated
		if slices.Contains(strings.Split(route["ifname"], ","), interfaceName) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys, nil
}
