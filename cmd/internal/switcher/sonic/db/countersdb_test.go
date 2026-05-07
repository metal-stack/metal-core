package db

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

func TestCountersDB_GetPortNameMap(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		want map[string]OID
	}{
		{
			name: "empty",
			data: test.StringMap{},
			want: map[string]OID{},
		},
		{
			name: "get port map",
			data: test.StringMap{
				"COUNTERS_PORT_NAME_MAP": test.StringMap{
					"Ethernet0": "oid:0x1000000000020",
					"Ethernet1": "oid:0x1000000000021",
					"Ethernet2": "oid:0x1000000000022",
					"Ethernet3": "oid:0x1000000000023",
				},
			},
			want: map[string]OID{
				"Ethernet0": "oid:0x1000000000020",
				"Ethernet1": "oid:0x1000000000021",
				"Ethernet2": "oid:0x1000000000022",
				"Ethernet3": "oid:0x1000000000023",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = ":"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &CountersDB{
				c: c,
			}
			got, err := d.GetPortNameMap(ctx)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("CountersDB.GetPortNameMap() diff = %s", diff)
			}
		})
	}
}

func TestCountersDB_GetRifNameMap(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		want map[string]OID
	}{
		{
			name: "empty",
			data: test.StringMap{},
			want: map[string]OID{},
		},
		{
			name: "get port map",
			data: test.StringMap{
				"COUNTERS_RIF_NAME_MAP": test.StringMap{
					"Ethernet0": "oid:0x1000000000020",
					"Ethernet1": "oid:0x1000000000021",
					"Ethernet2": "oid:0x1000000000022",
					"Ethernet3": "oid:0x1000000000023",
				},
			},
			want: map[string]OID{
				"Ethernet0": "oid:0x1000000000020",
				"Ethernet1": "oid:0x1000000000021",
				"Ethernet2": "oid:0x1000000000022",
				"Ethernet3": "oid:0x1000000000023",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = ":"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, tt.data, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &CountersDB{
				c: c,
			}
			got, err := d.GetRifNameMap(ctx)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("CountersDB.GetRifNameMap() diff = %s", diff)
			}
		})
	}
}
