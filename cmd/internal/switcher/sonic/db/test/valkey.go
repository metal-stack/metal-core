package test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

type (
	fields map[string]string
	obj    map[string]fields
	data   map[string]obj
)

func StartValkey(t testing.TB) valkey.Client {
	mr := miniredis.RunT(t)
	vc, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{mr.Addr()},
		// ClientOption.DisableCache must be true for valkey not supporting client-side caching or not supporting RESP3
		DisableCache: true,
	})
	require.NoError(t, err)
	return vc
}

func LoadData(ctx context.Context, vc valkey.Client, d data, separator string) error {
	for key, o := range d {
		for subkey, f := range o {
			if len(f) == 0 {
				// this is how SONiC adds empty maps
				err := hset(ctx, vc, strings.Join([]string{key, subkey}, separator), "NULL", "NULL")
				if err != nil {
					return err
				}
				continue
			}
			for field, value := range f {
				err := hset(ctx, vc, strings.Join([]string{key, subkey}, separator), field, value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func GetData(ctx context.Context, vc valkey.Client, separator string) (data, error) {
	d := data{}

	cmd := vc.B().Keys().Pattern("*").Build()
	res := vc.Do(ctx, cmd)
	if err := res.Error(); err != nil {
		return nil, err
	}

	keys, err := res.AsStrSlice()
	if err != nil {
		return nil, err
	}

	for _, k := range keys {
		cmd := vc.B().Hgetall().Key(k).Build()
		res := vc.Do(ctx, cmd)
		if err := res.Error(); err != nil {
			return nil, err
		}

		m, err := res.AsStrMap()
		if err != nil {
			return nil, err
		}

		key, subkey, found := strings.Cut(k, separator)
		if !found {
			return nil, fmt.Errorf("key %s does not contain the expected separator", k)
		}

		if d[key] == nil {
			d[key] = obj{}
		}
		d[key][subkey] = fields{}

		for f, v := range m {
			if f == "NULL" || v == "NULL" {
				continue
			}
			d[key][subkey][f] = v
		}
	}

	return d, nil
}

func hset(ctx context.Context, vc valkey.Client, key, field, value string) error {
	cmd := vc.B().Hset().Key(key).FieldValue().FieldValue(field, value).Build()
	return vc.Do(ctx, cmd).Error()
}
