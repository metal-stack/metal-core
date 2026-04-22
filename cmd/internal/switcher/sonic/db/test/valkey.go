package test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

type (
	stringMap map[string]any
	hashMap   map[string]map[string]string

	keysAndValue struct {
		keys  []string
		value string
	}
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

func LoadData(ctx context.Context, vc valkey.Client, data stringMap, separator string) error {
	kvs := getKeysAndValues(data)
	hm := getHashMap(kvs, separator)
	for k, m := range hm {
		if len(m) == 0 {
			err := hset(ctx, vc, k, "NULL", "NULL")
			if err != nil {
				return err
			}
			continue
		}
		for field, value := range m {
			err := hset(ctx, vc, k, field, value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getHashMap(kvs []keysAndValue, separator string) hashMap {
	m := hashMap{}
	for _, kv := range kvs {
		idx := len(kv.keys) - 1
		key := strings.Join(kv.keys[:idx], separator)
		if len(kv.keys) <= 2 {
			key += separator + kv.keys[idx]
			m[key] = map[string]string{}
			continue
		}
		if m[key] == nil {
			m[key] = map[string]string{}
		}
		m[key][kv.keys[idx]] = kv.value
	}
	return m
}

func getKeysAndValues(data stringMap) []keysAndValue {
	var keysAndValues []keysAndValue
	for k, v := range data {
		kv := keysAndValue{}
		switch v := v.(type) {
		case string:
			kv.keys = append(kv.keys, k)
			kv.value = v
			keysAndValues = append(keysAndValues, kv)
		case stringMap:
			if len(v) == 0 {
				keysAndValues = append(keysAndValues, keysAndValue{
					keys:  []string{k},
					value: "",
				})
				continue
			}
			kvs := getKeysAndValues(v)
			for i, kv := range kvs {
				kv.keys = slices.Concat([]string{k}, kv.keys)
				kvs[i] = kv
			}
			keysAndValues = append(keysAndValues, kvs...)
		}
	}
	return keysAndValues
}

func hset(ctx context.Context, vc valkey.Client, key, field, value string) error {
	cmd := vc.B().Hset().Key(key).FieldValue().FieldValue(field, value).Build()
	return vc.Do(ctx, cmd).Error()
}
