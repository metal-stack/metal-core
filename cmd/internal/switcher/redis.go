package switcher

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

const (
	configDB = "CONFIG_DB"
	stateDB  = "STATE_DB"
)

type SonicDatabaseConfig struct {
	Instances map[string]Instance `json:"INSTANCES"`
	Databases map[string]Database `json:"DATABASES"`
	Version   string              `json:"VERSION"`
}

type Instance struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
}

type Database struct {
	Id        int    `json:"id"`
	Separator string `json:"separator"`
	Instance  string `json:"instance"`
}

type ConfigDB struct {
	rdb       *redis.Client
	separator string
}

func NewConfigDB(cfg *SonicDatabaseConfig) *ConfigDB {
	db := cfg.Databases[configDB]
	i := cfg.Instances[db.Instance]
	rdb := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(i.Hostname, strconv.Itoa(i.Port)),
		DB:   db.Id,
	})
	return &ConfigDB{
		rdb:       rdb,
		separator: db.Separator,
	}
}

func (c *ConfigDB) SetEntry(key []string, values ...string) error {
	k := strings.Join(key, c.separator)
	if values == nil {
		return c.rdb.HSet(context.Background(), k, "NULL", "NULL").Err()
	}
	return c.rdb.HSet(context.Background(), k, values).Err()
}

func (c *ConfigDB) GetEntry(key []string) (map[string]string, error) {
	k := strings.Join(key, c.separator)
	return c.rdb.HGetAll(context.Background(), k).Result()
}

func (c *ConfigDB) DeleteEntry(key []string) error {
	k := strings.Join(key, c.separator)
	return c.rdb.Del(context.Background(), k).Err()
}

type View struct {
	keys      map[string]bool
	rdb       *redis.Client
	separator string
	table     string
}

func (c *ConfigDB) GetView(table string) (*View, error) {
	p := table + c.separator
	keys, err := c.rdb.Keys(context.Background(), p+"*").Result()
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(keys))
	for _, key := range keys {
		set[key] = false
	}
	return &View{
		keys:      set,
		rdb:       c.rdb,
		separator: c.separator,
		table:     table,
	}, nil
}

func (v *View) Contains(key []string) bool {
	k := v.table + v.separator + strings.Join(key, v.separator)
	_, ok := v.keys[k]
	return ok
}

func (v *View) Mask(key []string) {
	k := v.table + v.separator + strings.Join(key, v.separator)
	if _, ok := v.keys[k]; ok {
		v.keys[k] = true
	}
}

func (v *View) DeleteUnmasked() error {
	keys := make(map[string]bool)
	for key, masked := range v.keys {
		if masked {
			keys[key] = true
			continue
		}
		err := v.rdb.Del(context.Background(), key).Err()
		if err != nil {
			return err
		}
	}
	v.keys = keys
	return nil
}

type ConfigDBApplier struct {
	db *ConfigDB
}

func NewConfigDBApplier(cfg *SonicDatabaseConfig) *ConfigDBApplier {
	return &ConfigDBApplier{NewConfigDB(cfg)}
}

func (a *ConfigDBApplier) Apply(cfg *Conf) error {
	err := configureVxlan(a.db, cfg.Loopback)
	if err != nil {
		return err
	}
	err = applyVlan4000(a.db, cfg.MetalCoreCIDR)
	if err != nil {
		return err
	}
	return applyLoopback(a.db, cfg.Loopback)
}

func configureVxlan(db *ConfigDB, ip string) error {
	key := []string{"VXLAN_TUNNEL", "vtep"}
	entry, err := db.GetEntry(key)
	if err == redis.Nil {
		return db.SetEntry(key, "src_ip", ip)
	}
	if err != nil {
		return err
	}
	if entry["src_ip"] != ip {
		return db.SetEntry(key, "src_ip", ip)
	}
	return nil
}

func applyVlan4000(db *ConfigDB, cidr string) error {
	view, err := db.GetView("VLAN_INTERFACE")
	if err != nil {
		return err
	}

	infKey := []string{"VLAN_INTERFACE", "Vlan4000"}
	ipKey := []string{"VLAN_INTERFACE", "Vlan4000", cidr}
	if !view.Contains(infKey) {
		err = db.SetEntry(infKey)
		if err != nil {
			return err
		}
	}
	if !view.Contains(ipKey) {
		err = db.SetEntry(ipKey)
		if err != nil {
			return err
		}
	}

	view.Mask(infKey)
	view.Mask(ipKey)
	return view.DeleteUnmasked()
}

func applyLoopback(db *ConfigDB, ip string) error {
	view, err := db.GetView("LOOPBACK_INTERFACE")
	if err != nil {
		return err
	}

	infKey := []string{"LOOPBACK_INTERFACE", "Loopback0"}
	ipKey := []string{"LOOPBACK_INTERFACE", "Loopback0", ip + "/32"}
	if !view.Contains(infKey) {
		err = db.SetEntry(infKey)
		if err != nil {
			return err
		}
	}
	if !view.Contains(ipKey) {
		err = db.SetEntry(ipKey)
		if err != nil {
			return err
		}
	}

	view.Mask(infKey)
	view.Mask(ipKey)
	return view.DeleteUnmasked()
}
