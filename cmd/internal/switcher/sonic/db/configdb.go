package db

import (
	"context"
	"log/slog"

	"github.com/valkey-io/valkey-go"
)

type (
	ConfigDB struct {
		c   *Client
		log *slog.Logger
	}

	Port struct {
		Name        string
		Alias       string
		AdminStatus string
		Mtu         string
	}

	// Entry is a CONFIG_DB entry with the fields it should have.
	Entry struct {
		Key Key
		Val Val
	}
)

const (
	adminStatusField = "admin_status"
	alias            = "alias"
	mtu              = "mtu"
	portTable        = "PORT"
	null             = "NULL"
)

func newConfigDB(log *slog.Logger, rdb valkey.Client, sep string) *ConfigDB {
	return &ConfigDB{
		log: log,
		c:   NewClient(rdb, sep),
	}
}

// MergeEntry adds or updates the fields in e.Val while leaving all other fields untouched.
// An empty value is a no-op.
func (d *ConfigDB) MergeEntry(ctx context.Context, e Entry) error {
	if len(e.Val) == 0 {
		return nil
	}

	current, err := d.c.HGetAll(ctx, e.Key)
	if err != nil {
		return err
	}

	for field, value := range e.Val {
		if v, ok := current[field]; !ok || v != value {
			d.log.Debug("Updating entry", "key", e.Key, "previous", v, "value", e.Val)
			if err := d.c.HSet(ctx, e.Key, e.Val); err != nil {
				return err
			}
			break
		}
	}

	// NULL is an internal placeholder for entries without fields. It must not
	// remain once the entry has real fields.
	if _, hasNull := current[null]; hasNull {
		if _, wantsNull := e.Val[null]; !wantsNull {
			d.log.Debug("Deleting entry", "key", e.Key, "previous", e.Val)
			return d.c.HDel(ctx, e.Key, null)
		}
	}
	return nil
}

// SetFields sets and deletes fields of the entry. An entry without fields gets the NULL placeholder of SONiC,
// which is removed once the entry has fields.
func (d *ConfigDB) SetFields(ctx context.Context, key Key, set Val, del []string) error {
	if len(set) == 0 && len(del) == 0 {
		return d.c.HSet(ctx, key, Val{null: null})
	}
	if err := d.c.HSet(ctx, key, set); err != nil {
		return err
	}
	if len(set) > 0 {
		del = append(del, null)
	}
	return d.c.HDel(ctx, key, del...)
}

// DeleteEntry deletes the entry and reports whether it existed.
func (d *ConfigDB) DeleteEntry(ctx context.Context, key Key) (bool, error) {
	exists, err := d.c.Exists(ctx, key)
	if err != nil || !exists {
		return false, err
	}
	d.log.Debug("Deleted Entry", "key", key)
	return true, d.c.Del(ctx, key)
}

func (d *ConfigDB) GetPorts(ctx context.Context) ([]*Port, error) {
	var (
		ports     []*Port
		portNames []string
	)

	t := d.c.GetTable(Key{portTable})
	res, err := t.GetView(ctx)
	if err != nil {
		return nil, err
	}

	for name := range res {
		portNames = append(portNames, name)
	}

	for _, p := range portNames {
		result, err := d.c.HGetAll(ctx, Key{portTable, p})
		if err != nil {
			return nil, err
		}

		ports = append(ports, &Port{
			Name:        p,
			Alias:       result[alias],
			AdminStatus: result[adminStatusField],
			Mtu:         result[mtu],
		})
	}

	return ports, nil
}

// Dump reads all entries of the tables from the CONFIG_DB.
func (d *ConfigDB) Dump(ctx context.Context, tables ...Key) (Dump, error) {
	return d.c.Dump(ctx, tables...)
}
