package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeycompat"
)

type Key []string
type Val map[string]string

func (k *Key) toString(sep string) string {
	return strings.Join(*k, sep)
}

type Client struct {
	rdb valkey.Client
	sep string
}

func NewClient(rdb valkey.Client, sep string) *Client {
	return &Client{
		rdb: rdb,
		sep: sep,
	}
}

func (c *Client) Del(ctx context.Context, key Key) error {
	return c.rdb.Do(ctx, c.rdb.B().Del().Key(key.toString(c.sep)).Build()).Error()
}

func (c *Client) Exists(ctx context.Context, key Key) (bool, error) {
	result, err := c.rdb.Do(ctx, c.rdb.B().Exists().Key(key.toString(c.sep)).Build()).AsBool()
	if err != nil {
		return false, err
	}
	return result, nil
}

func (c *Client) GetTable(table Key) *Table {
	return &Table{
		client: c,
		name:   table.toString(c.sep),
	}
}

func (c *Client) GetView(ctx context.Context, table string) (View, error) {
	var (
		prefix  = table + c.sep
		pattern = prefix + "*"
	)

	keys, err := c.rdb.Do(ctx, c.rdb.B().Keys().Pattern(pattern).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}

	view := NewView(len(keys))
	for _, key := range keys {
		item, found := strings.CutPrefix(key, prefix)
		if !found {
			return nil, fmt.Errorf("key %s does not contain expected prefix %s", key, prefix)
		}
		view.Add(item)
	}

	return view, nil
}

func (c *Client) HGet(ctx context.Context, key Key, field string) (string, error) {
	result, err := c.rdb.Do(ctx, c.rdb.B().Hget().Key(key.toString(c.sep)).Field(field).Build()).ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return "", nil
		}
		return "", err
	}
	return result, nil
}

func (c *Client) HGetAll(ctx context.Context, key Key) (Val, error) {
	return c.rdb.Do(ctx, c.rdb.B().Hgetall().Key(key.toString(c.sep)).Build()).AsStrMap()
}

func (c *Client) HSet(ctx context.Context, key Key, val Val) error {
	compat := valkeycompat.NewAdapter(c.rdb)
	// FIXME migrate to native
	return compat.HSet(ctx, key.toString(c.sep), map[string]string(val)).Err()
}

func (c *Client) Keys(ctx context.Context, pattern Key) ([]Key, error) {
	result, err := c.rdb.Do(ctx, c.rdb.B().Keys().Pattern(pattern.toString(c.sep)).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}

	keys := make([]Key, 0, len(result))
	for _, item := range result {
		key := strings.Split(item, c.sep)
		keys = append(keys, key)
	}
	return keys, nil
}
