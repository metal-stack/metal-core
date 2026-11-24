package db

import (
	"context"

	"github.com/valkey-io/valkey-go"
)

type (
	AsicDB struct {
		c *Client
	}

	OID string
)

func newAsicDB(rdb valkey.Client, sep string) *AsicDB {
	return &AsicDB{
		c: NewClient(rdb, sep),
	}
}

// Dump reads all entries of the tables from the ASIC_DB.
func (d *AsicDB) Dump(ctx context.Context, tables ...Key) (Dump, error) {
	return d.c.Dump(ctx, tables...)
}
