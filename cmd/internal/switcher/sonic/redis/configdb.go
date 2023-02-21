package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	enable          = "enable"
	interfaceTable  = "INTERFACE"
	linkLocalOnly   = "ipv6_use_link_local_only"
	vlanMemberTable = "VLAN_MEMBER"
	taggingMode     = "tagging_mode"
	untagged        = "untagged"
	vrfName         = "vrf_name"
)

type configdb struct {
	log       *zap.SugaredLogger
	rdb       *redis.Client
	separator string
}

func newConfigdb(log *zap.SugaredLogger, addr string, id int, separator string) *configdb {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       id,
		PoolSize: 1,
	})
	return &configdb{
		log:       log,
		rdb:       rdb,
		separator: separator,
	}
}

func (c *configdb) getVlanMembership(ctx context.Context, interfaceName string) ([]string, error) {
	pattern := vlanMemberTable + c.separator + "*" + c.separator + interfaceName

	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	vlans := make([]string, 0, len(keys))
	for _, key := range keys {
		split := strings.Split(key, c.separator)
		if len(split) != 3 {
			return nil, fmt.Errorf("could not parse key %s", key)
		}
		vlans = append(vlans, split[1])
	}
	return vlans, nil
}

func (c *configdb) setVlanMember(ctx context.Context, interfaceName, vlan string) error {
	key := vlanMemberTable + c.separator + vlan + c.separator + interfaceName

	return c.rdb.HSet(ctx, key, taggingMode, untagged).Err()
}

func (c *configdb) deleteVlanMember(ctx context.Context, interfaceName, vlan string) error {
	key := vlanMemberTable + c.separator + vlan + c.separator + interfaceName

	return c.rdb.Del(ctx, key).Err()
}

func (c *configdb) setVrfMember(ctx context.Context, interfaceName string, vrf string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.HSet(ctx, key, linkLocalOnly, enable, vrfName, vrf).Err()
}

func (c *configdb) getVrfMembership(ctx context.Context, interfaceName string) (string, error) {
	key := interfaceTable + c.separator + interfaceName

	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return result[vrfName], nil
}

func (c *configdb) deleteVrfMember(ctx context.Context, interfaceName string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.Del(ctx, key).Err()
}

func (c *configdb) isLinkLocalOnly(ctx context.Context, interfaceName string) (bool, error) {
	key := interfaceTable + c.separator + interfaceName

	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result[linkLocalOnly] == enable, nil
}

func (c *configdb) enableLinkLocalOnly(ctx context.Context, interfaceName string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.HSet(ctx, key, linkLocalOnly, enable).Err()
}

func (c *configdb) disableLinkLocalOnly(ctx context.Context, interfaceName string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.HDel(ctx, key, linkLocalOnly).Err()
}
