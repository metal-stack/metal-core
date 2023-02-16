package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type configdb struct {
	rdb       *redis.Client
	separator string
}

func newConfigdb(addr string, id int, separator string) *configdb {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   id,
	})
	return &configdb{
		rdb:       rdb,
		separator: separator,
	}
}

const interfaceTable = "INTERFACE"
const vlanMemberTable = "VLAN_MEMBER"

func (c *configdb) setVlanMember(ctx context.Context, interfaceName, vlan string) error {
	key := vlanMemberTable + c.separator + vlan + c.separator + interfaceName

	return c.rdb.HSet(ctx, key, "tagging_mode", "untagged").Err()
}

func (c *configdb) deleteVlanMember(ctx context.Context, interfaceName string, vlan uint16) error {
	key := vlanMemberTable + c.separator + "Vlan" + fmt.Sprintf("%d", vlan) + c.separator + interfaceName

	return c.rdb.Del(ctx, key).Err()
}

func (c *configdb) setVrfMember(ctx context.Context, interfaceName string, vrf string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.HSet(ctx, key, "vrf_name", vrf).Err()
}

func (c *configdb) deleteVrfMember(ctx context.Context, interfaceName string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.HDel(ctx, key, "vrf_name").Err()
}
