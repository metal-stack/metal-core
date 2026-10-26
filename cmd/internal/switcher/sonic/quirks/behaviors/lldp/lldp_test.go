package lldp

import (
	"context"
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

// fakeConfigDB keeps the CONFIG_DB as map of "TABLE|key" to fields
type fakeConfigDB struct {
	entries map[string]db.Val
	err     error
}

func (f *fakeConfigDB) EnsureEntry(_ context.Context, e db.Entry) error {
	if f.err != nil {
		return f.err
	}
	key := strings.Join(e.Key, "|")
	if f.entries[key] == nil {
		f.entries[key] = db.Val{}
	}
	maps.Copy(f.entries[key], e.Val)
	return nil
}

func (f *fakeConfigDB) DeleteEntry(_ context.Context, key db.Key) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	k := strings.Join(key, "|")
	_, ok := f.entries[k]
	delete(f.entries, k)
	return ok, nil
}

func TestLLDPConfiguration_Configure(t *testing.T) {
	ports := []*db.Port{
		{Name: "Ethernet0", Alias: "Eth1/1"},
		{Name: "Ethernet4", Alias: "Eth1/2"},
	}

	tests := []struct {
		name     string
		lldp     LLDPConfiguration
		existing map[string]db.Val
		want     map[string]db.Val
	}{
		{
			name: "community removes the port config to get the tx-rx default",
			lldp: Community{},
			existing: map[string]db.Val{
				"LLDP_PORT|Ethernet0": {"enabled": "true", "mode": "RECEIVE"},
				"LLDP_PORT|Ethernet8": {"enabled": "true"},
			},
			want: map[string]db.Val{
				"LLDP|GLOBAL":         {"hello_time": "10"},
				"LLDP_PORT|Ethernet8": {"enabled": "true"},
			},
		},
		{
			name: "asternos sets the port id",
			lldp: AsterNOS{},
			existing: map[string]db.Val{
				"LLDP|Ethernet0": {"lldp_local_portid": "Ethernet0", "other": "kept"},
			},
			want: map[string]db.Val{
				"LLDP|lldp":      {"tx_interval": "10"},
				"LLDP|Ethernet0": {"lldp_local_portid": "Eth1/1", "other": "kept"},
				"LLDP|Ethernet4": {"lldp_local_portid": "Eth1/2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &fakeConfigDB{entries: tt.existing}
			require.Empty(t, tt.lldp.Configure(context.Background(), c, ports))
			if diff := cmp.Diff(tt.want, c.entries); diff != "" {
				t.Errorf("Configure() diff = %s", diff)
			}
		})
	}
}

func TestLLDPConfiguration_ConfigureErrors(t *testing.T) {
	ports := []*db.Port{{Name: "Ethernet0"}, {Name: "Ethernet4"}}

	for _, lldp := range []LLDPConfiguration{Community{}, AsterNOS{}} {
		c := &fakeConfigDB{err: errors.New("valkey down")}
		// one error for the tx_interval and one per port, so a failing port does not hide the others
		require.Len(t, lldp.Configure(context.Background(), c, ports), 3)
	}
}
