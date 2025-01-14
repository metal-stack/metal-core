package db

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

type AsicDB struct {
	c *Client
}

type OID string

func newAsicDB(rdb *redis.Client, sep string) *AsicDB {
	return &AsicDB{
		c: NewClient(rdb, sep),
	}
}

func (d *AsicDB) GetPortIdBridgePortMap(ctx context.Context) (map[OID]OID, error) {
	t := d.c.GetTable(Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT"})

	bridges, err := t.GetView(ctx)
	if err != nil {
		return nil, err
	}

	m := make(map[OID]OID, len(bridges))
	for bridge := range bridges {
		port, err := t.HGet(ctx, bridge, "SAI_BRIDGE_PORT_ATTR_PORT_ID")
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return nil, err
		}
		if len(port) == 0 {
			continue
		}
		m[OID(port)] = OID(bridge)
	}
	return m, nil
}

func (d *AsicDB) ExistBridgePort(ctx context.Context, bridgePort OID) (bool, error) {
	key := Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", string(bridgePort)}

	return d.c.Exists(ctx, key)
}

func (d *AsicDB) ExistRouterInterface(ctx context.Context, rif OID) (bool, error) {
	key := Key{"ASIC_STATE", "SAI_OBJECT_TYPE_ROUTER_INTERFACE", string(rif)}

	return d.c.Exists(ctx, key)
}
