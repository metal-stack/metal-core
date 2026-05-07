package test

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

type (
	StringMap map[string]any
	HashMap   map[string]map[string]string

	keysAndValue struct {
		keys  []string
		value string
	}
)

const (
	null = "NULL"
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

func LoadData(ctx context.Context, vc valkey.Client, data StringMap, separator string) error {
	kvs := getKeysAndValues(data)
	hm := getHashMap(kvs, separator)
	for k, m := range hm {
		if len(m) == 0 {
			err := hset(ctx, vc, k, null, null)
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

func GetData(ctx context.Context, vc valkey.Client, separator string) (HashMap, error) {
	cmd := vc.B().Keys().Pattern("*").Build()
	res := vc.Do(ctx, cmd)
	if err := res.Error(); err != nil {
		return nil, err
	}
	keys, err := res.AsStrSlice()
	if err != nil {
		return nil, err
	}
	hm := HashMap{}
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
		if hm[k] == nil {
			hm[k] = map[string]string{}
		}
		maps.Copy(hm[k], m)
	}
	return hm, nil
}

func cutPrefixFromHashMap(hm HashMap, prefix string) HashMap {
	if prefix == "" {
		return hm
	}
	m := HashMap{}
	for k, v := range hm {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		newKey := strings.TrimPrefix(k, prefix)
		m[newKey] = v
	}
	return m
}

func getHashMap(kvs []keysAndValue, separator string) HashMap {
	m := HashMap{}
	for _, kv := range kvs {
		idx := len(kv.keys) - 1
		key := strings.Join(kv.keys[:idx], separator)
		if kv.value == null {
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

func getKeysAndValues(data StringMap) []keysAndValue {
	var keysAndValues []keysAndValue
	for k, v := range data {
		kv := keysAndValue{}
		switch v := v.(type) {
		case string:
			kv.keys = append(kv.keys, k)
			kv.value = v
			keysAndValues = append(keysAndValues, kv)
		case StringMap:
			if len(v) == 0 {
				keysAndValues = append(keysAndValues, keysAndValue{
					keys:  []string{k},
					value: null,
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
