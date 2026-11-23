package db

import (
	"context"
	"fmt"
	"maps"
	"slices"

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

// GetRoutes returns the routes fpmsyncd installed in the vrf, sorted by key. An empty vrf returns the routes of all vrfs.
//
// Next hop groups are resolved, so the next hops of a route are complete. A route whose next hop group does not exist
// is an error instead of a route without next hops, which would look like a route that uses no interface.
func (d *ApplDB) GetRoutes(ctx context.Context, vrf string) ([]Route, error) {
	pattern := Key{"ROUTE_TABLE", vrf, "*"}
	if vrf == "" {
		pattern = Key{"ROUTE_TABLE", "*"}
	}

	entries, err := d.c.HGetAllMatching(ctx, pattern)
	if err != nil {
		return nil, err
	}

	keys := slices.Sorted(maps.Keys(entries))
	routes := make([]Route, 0, len(keys))
	for _, key := range keys {
		val := entries[key]

		var group Val
		if id := val["nexthop_group"]; id != "" {
			group, err = d.c.HGetAll(ctx, Key{"NEXTHOP_GROUP_TABLE", id})
			if err != nil {
				return nil, fmt.Errorf("could not retrieve next hop group %s of route %s: %w", id, key, err)
			}
			if len(group) == 0 {
				return nil, fmt.Errorf("next hop group %s of route %s does not exist", id, key)
			}
		}

		route, err := parseRoute(key, val, group)
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, nil
}
