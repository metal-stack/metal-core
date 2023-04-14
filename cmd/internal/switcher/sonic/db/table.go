package db

import (
	"context"
)

type Table struct {
	client *Client
	name   string
}

func (t *Table) Del(ctx context.Context, item string) error {
	return t.client.Del(ctx, Key{t.name, item})
}

func (t *Table) Exists(ctx context.Context, item string) (bool, error) {
	return t.client.Exists(ctx, Key{t.name, item})
}

func (t *Table) GetView(ctx context.Context) (View, error) {
	return t.client.GetView(ctx, t.name)
}

func (t *Table) HGet(ctx context.Context, item string, field string) (string, error) {
	return t.client.HGet(ctx, Key{t.name, item}, field)
}

func (t *Table) HGetAll(ctx context.Context, item string) (Val, error) {
	return t.client.HGetAll(ctx, Key{t.name, item})
}

func (t *Table) HSet(ctx context.Context, item string, val Val) error {
	return t.client.HSet(ctx, Key{t.name, item}, val)
}
