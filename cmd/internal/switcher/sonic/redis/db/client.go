package db

import (
	"context"
	"errors"
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

func (d *Client) Del(ctx context.Context, key Key) error {
	return d.rdb.Del(ctx, key.toString(d.sep)).Err()
}

func (d *Client) Exists(ctx context.Context, key Key) (bool, error) {
	result, err := d.rdb.Exists(ctx, key.toString(d.sep)).Result()
	if err != nil {
		return false, err
	}
	return result != 0, nil
}

func (d *Client) HGet(ctx context.Context, key Key, field string) (string, error) {
	result, err := d.rdb.HGet(ctx, key.toString(d.sep), field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return result, nil
}

func (d *Client) HGetAll(ctx context.Context, key Key) (Val, error) {
	return d.rdb.HGetAll(ctx, key.toString(d.sep)).Result()
}

func (d *Client) HSet(ctx context.Context, key Key, val Val) error {
	return d.rdb.HSet(ctx, key.toString(d.sep), map[string]string(val)).Err()
}

func (d *Client) Keys(ctx context.Context, pattern Key) ([]Key, error) {
	result, err := d.rdb.Keys(ctx, pattern.toString(d.sep)).Result()
	if err != nil {
		return nil, err
	}

	keys := make([]Key, 0, len(result))
	for _, item := range result {
		key := strings.Split(item, d.sep)
		keys = append(keys, key)
	}
	return keys, nil
}
