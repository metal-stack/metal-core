package db

import (
	"context"
	"errors"
	"strings"

	"github.com/redis/go-redis/v9"
)

type AsicDB struct {
	c *Client
}

type OID string

func newAsicDB(addr string, id int, sep string) *AsicDB {
	return &AsicDB{
		c: NewClient(addr, id, sep),
	}
}

func (d *AsicDB) GetPortIdBridgePortMap(ctx context.Context) (map[OID]OID, error) {
	pattern := Key{"ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT", "*"}

	bridges, err := d.c.rdb.Keys(ctx, pattern.toString(d.c.sep)).Result()
	if err != nil {
		return nil, err
	}

	m := make(map[OID]OID, len(bridges))
	for _, bridge := range bridges {
		port, err := d.c.rdb.HGet(ctx, bridge, "SAI_BRIDGE_PORT_ATTR_PORT_ID").Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return nil, err
		}
		if len(port) == 0 {
			continue
		}
		split := strings.SplitN(bridge, d.c.sep, 3)
		if len(split) == 3 && strings.HasPrefix(split[2], "oid:") {
			m[OID(port)] = OID(split[2])
		}
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

func (d *AsicDB) InFecModeRs(ctx context.Context, port OID) (bool, error) {
	key := Key{"ASIC_STATE", "SAI_OBJECT_TYPE_PORT", string(port)}

	result, err := d.c.HGet(ctx, key, "SAI_PORT_ATTR_FEC_MODE")
	if err != nil {
		return false, err
	}
	return result == "SAI_PORT_FEC_MODE_RS", err
}
