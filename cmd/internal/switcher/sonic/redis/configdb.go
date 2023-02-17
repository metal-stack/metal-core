package redis

import (
	"context"
	"errors"
	"fmt"

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

func (c *configdb) setVlanMember(ctx context.Context, interfaceName, vlan string) error {
	key := vlanMemberTable + c.separator + vlan + c.separator + interfaceName

	// If the key doesn't exist then an empty map will be returned instead of an error
	// https://github.com/redis/go-redis/issues/1668#issuecomment-781090968
	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return err
	}
	if len(result) == 1 && result[taggingMode] == untagged {
		return nil
	}

	return c.rdb.HSet(ctx, key, taggingMode, untagged).Err()
}

func (c *configdb) deleteVlanMember(ctx context.Context, interfaceName string, vlan uint16) error {
	key := vlanMemberTable + c.separator + "Vlan" + fmt.Sprintf("%d", vlan) + c.separator + interfaceName

	err := c.rdb.Get(ctx, key).Err()
	if !errors.Is(err, redis.Nil) {
		return nil
	} else if err != nil {
		return err
	}

	return c.rdb.Del(ctx, key).Err()
}

func (c *configdb) setVrfMember(ctx context.Context, interfaceName string, vrf string) error {
	key := interfaceTable + c.separator + interfaceName

	// If the key doesn't exist then an empty map will be returned instead of an error
	// https://github.com/redis/go-redis/issues/1668#issuecomment-781090968
	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return err
	}
	if len(result) == 2 && result[linkLocalOnly] == enable && result[vrfName] == vrf {
		return nil
	}

	return c.rdb.HSet(ctx, key, linkLocalOnly, enable, vrfName, vrf).Err()
}

func (c *configdb) deleteVrfMember(ctx context.Context, interfaceName string) error {
	key := interfaceTable + c.separator + interfaceName

	// If the key doesn't exist then an empty map will be returned instead of an error
	// https://github.com/redis/go-redis/issues/1668#issuecomment-781090968
	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return err
	}
	if _, inVrf := result[vrfName]; len(result) == 2 && result[linkLocalOnly] == enable && inVrf {
		return c.rdb.HDel(ctx, key, linkLocalOnly, vrfName).Err()
	}

	return nil
}

func (c *configdb) enableLinkLocalOnly(ctx context.Context, interfaceName string) error {
	key := interfaceTable + c.separator + interfaceName

	// If the key doesn't exist then an empty map will be returned instead of an error
	// https://github.com/redis/go-redis/issues/1668#issuecomment-781090968
	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return err
	}
	if result[linkLocalOnly] == enable {
		return nil
	}

	return c.rdb.HSet(ctx, key, linkLocalOnly, enable).Err()
}
