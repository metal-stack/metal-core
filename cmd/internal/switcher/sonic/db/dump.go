package db

import (
	"context"
	"maps"
	"slices"
	"strings"
)

// Dump holds the entries of whole tables of one database, as they were read at one point in time.
type Dump struct {
	sep     string
	entries map[string]Val
}

// NewDump returns a dump of the entries, which are keyed like in the database.
func NewDump(sep string, entries map[string]Val) Dump {
	if entries == nil {
		entries = map[string]Val{}
	}
	return Dump{sep: sep, entries: entries}
}

// Get returns the fields of the entry. The NULL placeholder of entries without fields is left out.
func (d Dump) Get(key Key) (Val, bool) {
	val, ok := d.entries[key.toString(d.sep)]
	if !ok {
		return nil, false
	}
	fields := maps.Clone(val)
	delete(fields, null)
	return fields, true
}

func (d Dump) Has(key Key) bool {
	_, ok := d.entries[key.toString(d.sep)]
	return ok
}

// Keys returns the sorted keys of the entries in the table, split by the separator.
// It must only be used for tables whose keys do not contain the separator otherwise, unlike the ROUTE_TABLE.
func (d Dump) Keys(table Key) []Key {
	prefix := table.toString(d.sep) + d.sep

	var keys []Key
	for _, k := range slices.Sorted(maps.Keys(d.entries)) {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, strings.Split(k, d.sep))
		}
	}
	return keys
}

// Items returns the sorted keys of the entries in the table without the table, unsplit.
// Unlike Keys, it can be used for tables whose keys contain the separator, like the oids in the ASIC_DB.
func (d Dump) Items(table Key) []string {
	prefix := table.toString(d.sep) + d.sep

	var items []string
	for _, k := range slices.Sorted(maps.Keys(d.entries)) {
		if item, ok := strings.CutPrefix(k, prefix); ok {
			items = append(items, item)
		}
	}
	return items
}

// String returns the key as it is stored in the database.
func (d Dump) String(key Key) string {
	return key.toString(d.sep)
}

// Dump reads all entries of the tables.
func (c *Client) Dump(ctx context.Context, tables ...Key) (Dump, error) {
	entries := map[string]Val{}
	for _, table := range tables {
		pattern := append(slices.Clone(table), "*")
		matching, err := c.HGetAllMatching(ctx, pattern)
		if err != nil {
			return Dump{}, err
		}
		maps.Copy(entries, matching)
	}
	return NewDump(c.sep, entries), nil
}
