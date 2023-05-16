package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type Key []string
type Val map[string]string

func (k *Key) toString(sep string) string {
	return strings.Join(*k, sep)
}

type Client struct {
	rdb *redis.Client
	sep string
}

func NewClient(addr string, id int, sep string) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       id,
		PoolSize: 1,
	})
	return &Client{
		rdb: rdb,
		sep: sep,
	}
}

func (c *Client) Del(ctx context.Context, key Key) error {
	return c.rdb.Del(ctx, key.toString(c.sep)).Err()
}

func (c *Client) Exists(ctx context.Context, key Key) (bool, error) {
	result, err := c.rdb.Exists(ctx, key.toString(c.sep)).Result()
	if err != nil {
		return false, err
	}
	return result != 0, nil
}

func (c *Client) GetTable(table Key) *Table {
	return &Table{
		client: c,
		name:   table.toString(c.sep),
	}
}

func (c *Client) GetView(ctx context.Context, table string) (View, error) {
	pattern := table + c.sep + "*"
	keys, err := c.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	view := NewView(len(keys))
	for _, key := range keys {
		if len(key) < len(pattern) {
			return nil, fmt.Errorf("key %s is slower than pattern %s", key, pattern)
		}
		item := key[len(table)+1:]
		view.Add(item)
	}
	return view, nil
}

func (c *Client) HGet(ctx context.Context, key Key, field string) (string, error) {
	result, err := c.rdb.HGet(ctx, key.toString(c.sep), field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return result, nil
}

func (c *Client) HGetAll(ctx context.Context, key Key) (Val, error) {
	return c.rdb.HGetAll(ctx, key.toString(c.sep)).Result()
}

func (c *Client) HSet(ctx context.Context, key Key, val Val) error {
	return c.rdb.HSet(ctx, key.toString(c.sep), map[string]string(val)).Err()
}

func (c *Client) Keys(ctx context.Context, pattern Key) ([]Key, error) {
	result, err := c.rdb.Keys(ctx, pattern.toString(c.sep)).Result()
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
