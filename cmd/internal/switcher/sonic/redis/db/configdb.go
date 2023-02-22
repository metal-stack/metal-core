package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
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

type ConfigDB struct {
	rdb       *redis.Client
	separator string
}

func newConfigDB(addr string, id int, separator string) *ConfigDB {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       id,
		PoolSize: 1,
	})
	return &ConfigDB{
		rdb:       rdb,
		separator: separator,
	}
}

func (c *ConfigDB) GetVlanMembership(ctx context.Context, interfaceName string) ([]string, error) {
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

func (c *ConfigDB) SetVlanMember(ctx context.Context, interfaceName, vlan string) error {
	key := vlanMemberTable + c.separator + vlan + c.separator + interfaceName

	return c.rdb.HSet(ctx, key, taggingMode, untagged).Err()
}

func (c *ConfigDB) DeleteVlanMember(ctx context.Context, interfaceName, vlan string) error {
	key := vlanMemberTable + c.separator + vlan + c.separator + interfaceName

	return c.rdb.Del(ctx, key).Err()
}

func (c *ConfigDB) SetVrfMember(ctx context.Context, interfaceName string, vrf string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.HSet(ctx, key, vrfName, vrf).Err()
}

func (c *ConfigDB) GetVrfMembership(ctx context.Context, interfaceName string) (string, error) {
	key := interfaceTable + c.separator + interfaceName

	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return result[vrfName], nil
}

func (c *ConfigDB) DeleteInterfaceConfiguration(ctx context.Context, interfaceName string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.Del(ctx, key).Err()
}

func (c *ConfigDB) IsLinkLocalOnly(ctx context.Context, interfaceName string) (bool, error) {
	key := interfaceTable + c.separator + interfaceName

	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result[linkLocalOnly] == enable, nil
}

func (c *ConfigDB) EnableLinkLocalOnly(ctx context.Context, interfaceName string) error {
	key := interfaceTable + c.separator + interfaceName

	return c.rdb.HSet(ctx, key, linkLocalOnly, enable).Err()
}
