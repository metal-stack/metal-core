package configdb

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Options struct {
	Addr      string
	Id        int
	Separator string
}

type database struct {
	rdb       *redis.Client
	separator string
}

func newRedis(option *Options) *database {
	rdb := redis.NewClient(&redis.Options{
		Addr: option.Addr,
		DB:   option.Id,
	})
	return &database{
		rdb:       rdb,
		separator: option.Separator,
	}
}

const INTERFACE = "INTERFACE"
const VLAN_MEMBER_TABLE = "VLAN_MEMBER"

func (r *database) setVLANMember(ctx context.Context, interfaceName, vlan string) error {
	key := VLAN_MEMBER_TABLE + r.separator + vlan + r.separator + interfaceName

	return r.rdb.HSet(ctx, key, "tagging_mode", "untagged").Err()
}

func (r *database) deleteVLANMember(ctx context.Context, interfaceName string, vlan uint16) error {
	key := VLAN_MEMBER_TABLE + r.separator + "Vlan" + fmt.Sprintf("%d", vlan) + r.separator + interfaceName

	return r.rdb.Del(ctx, key).Err()
}

func (r *database) setVRFMember(ctx context.Context, interfaceName string, vrf string) error {
	key := INTERFACE + r.separator + interfaceName

	return r.rdb.HSet(ctx, key, "vrf_name", vrf).Err()
}

func (r *database) deleteVRFMember(ctx context.Context, interfaceName string) error {
	key := INTERFACE + r.separator + interfaceName

	return r.rdb.HSet(ctx, key, "NULL", "NULL").Err()
}
