package db

import (
	"context"
	"maps"
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

// GetRoutes returns the keys of the routes fpmsyncd installed in the vrf.
func (d *ApplDB) GetRoutes(ctx context.Context, vrf string) ([]string, error) {
	routes, err := d.c.HGetAllMatching(ctx, Key{"ROUTE_TABLE", vrf, "*"})
	if err != nil {
		return nil, err
	}

	keys := slices.Collect(maps.Keys(routes))
	slices.Sort(keys)
	return keys, nil
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
