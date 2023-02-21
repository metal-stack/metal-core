package redis

import (
	"context"
	"fmt"
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

	c.log.Infof("add interface %s to vlan %s", interfaceName, vlan)
	return c.rdb.HSet(ctx, key, taggingMode, untagged).Err()
}

func (c *configdb) deleteVlanMember(ctx context.Context, interfaceName string, vlan uint16) error {
	key := vlanMemberTable + c.separator + "Vlan" + fmt.Sprintf("%d", vlan) + c.separator + interfaceName

	result, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	} else if result == 0 {
		return nil
	}

	c.log.Infof("remove interface %s from vlan Vlan%d", interfaceName, vlan)
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

	c.log.Infof("add interface %s to vrf %s", interfaceName, vrfName)
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
	if vrf, inVrf := result[vrfName]; len(result) == 2 && result[linkLocalOnly] == enable && inVrf {
		c.log.Infof("remove interface %s from vrf %s", interfaceName, vrf)
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
