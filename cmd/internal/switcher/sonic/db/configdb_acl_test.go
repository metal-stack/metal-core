package db

import (
	"context"
	"slices"
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

func TestConfigDB_ACLTablePorts(t *testing.T) {
	setup := func(t *testing.T, ports string) (*ConfigDB, valkey.Client) {
		t.Helper()
		vc := test.StartValkey(t)
		t.Cleanup(vc.Close)
		require.NoError(t, test.LoadData(t.Context(), vc, test.StringMap{
			"ACL_TABLE": test.StringMap{
				"TENANT_V6": test.StringMap{"ports@": ports, "type": "L3V6", "stage": "ingress"},
				"TENANT_V4": test.StringMap{"type": "L3"},
			},
			"ACL_RULE": test.StringMap{
				"TENANT_V6|ALLOW": test.StringMap{"PACKET_ACTION": "FORWARD", "PRIORITY": "100"},
			},
		}, "|"))
		return newConfigDB(vc, "|"), vc
	}

	t.Run("list empty database", func(t *testing.T) {
		vc := test.StartValkey(t)
		t.Cleanup(vc.Close)
		tables, err := newConfigDB(vc, "|").GetACLTables(t.Context())
		require.NoError(t, err)
		require.Empty(t, tables)
	})

	t.Run("read tables and bindings", func(t *testing.T) {
		d, _ := setup(t, " Ethernet3, , Ethernet1 , ")
		ctx := t.Context()
		tables, err := d.GetACLTables(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"TENANT_V4", "TENANT_V6"}, tables)
		ports, err := d.GetACLTablePorts(ctx, "TENANT_V6")
		require.NoError(t, err)
		require.Equal(t, []string{"Ethernet1", "Ethernet3"}, ports)
		ports, err = d.GetACLTablePorts(ctx, "TENANT_V4")
		require.NoError(t, err)
		require.Empty(t, ports)
	})

	for _, tt := range []struct {
		name     string
		table    string
		existing string
		ports    []string
		delete   bool
		want     string
	}{
		{name: "add VRF ports", table: "TENANT_V6", existing: "Ethernet1,Ethernet3", ports: []string{"Ethernet4", "Ethernet2", "Ethernet4"}, want: "Ethernet1,Ethernet2,Ethernet3,Ethernet4"},
		{name: "add existing port", table: "TENANT_V6", existing: "Ethernet1,Ethernet3", ports: []string{"Ethernet1"}, want: "Ethernet1,Ethernet3"},
		{name: "add first binding", table: "TENANT_V4", existing: "Ethernet1,Ethernet3", ports: []string{"Ethernet2"}, want: "Ethernet2"},
		{name: "delete only this VRF ports", table: "TENANT_V6", existing: "Ethernet1,Ethernet2,Ethernet3,Ethernet4", delete: true, ports: []string{"Ethernet4", "Ethernet2", "Ethernet4"}, want: "Ethernet1,Ethernet3"},
		{name: "delete last port", table: "TENANT_V6", existing: "Ethernet3", delete: true, ports: []string{"Ethernet3"}, want: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d, vc := setup(t, tt.existing)
			ctx := t.Context()
			before, err := test.GetData(ctx, vc, "|")
			require.NoError(t, err)
			original := slices.Clone(tt.ports)
			if tt.delete {
				require.NoError(t, d.DeleteACLTablePorts(ctx, tt.table, tt.ports))
			} else {
				require.NoError(t, d.AddACLTablePorts(ctx, tt.table, tt.ports))
			}
			require.Equal(t, original, tt.ports, "caller slice must be preserved")
			before["ACL_TABLE|"+tt.table]["ports@"] = tt.want
			after, err := test.GetData(ctx, vc, "|")
			require.NoError(t, err)
			require.Equal(t, before, after, "other bindings, tables and rules must be preserved")
		})
	}

	for name, operation := range map[string]func(*ConfigDB, context.Context, string, []string) error{
		"add":    (*ConfigDB).AddACLTablePorts,
		"delete": (*ConfigDB).DeleteACLTablePorts,
	} {
		t.Run(name, func(t *testing.T) {
			for _, tt := range []struct {
				name  string
				ports []string
			}{
				{name: "empty name", ports: []string{""}},
				{name: "mixed valid and invalid names", ports: []string{"Ethernet1", ""}},
				{name: "comma in name", ports: []string{"Ethernet1,Ethernet3"}},
				{name: "surrounding whitespace", ports: []string{" Ethernet1 "}},
			} {
				t.Run(tt.name, func(t *testing.T) {
					d, vc := setup(t, "Ethernet1,Ethernet3")
					ctx := t.Context()
					before, err := test.GetData(ctx, vc, "|")
					require.NoError(t, err)
					require.Error(t, operation(d, ctx, "TENANT_V6", tt.ports))
					after, err := test.GetData(ctx, vc, "|")
					require.NoError(t, err)
					require.Equal(t, before, after, "invalid input must not change bindings")
				})
			}
			for _, table := range []string{"TENANT_V6", "TENANT_V4", "MISSING"} {
				t.Run("empty input for "+table, func(t *testing.T) {
					d, vc := setup(t, "Ethernet1,Ethernet3")
					ctx := t.Context()
					before, err := test.GetData(ctx, vc, "|")
					require.NoError(t, err)
					require.NoError(t, operation(d, ctx, table, nil))
					after, err := test.GetData(ctx, vc, "|")
					require.NoError(t, err)
					require.Equal(t, before, after, "empty input must not change or create bindings")
				})
			}
		})
	}

	for _, table := range []string{"TENANT_V6", "TENANT_V4", "MISSING"} {
		t.Run("delete absent binding from "+table, func(t *testing.T) {
			d, vc := setup(t, "Ethernet1,Ethernet3")
			ctx := t.Context()
			before, err := test.GetData(ctx, vc, "|")
			require.NoError(t, err)
			require.NoError(t, d.DeleteACLTablePorts(ctx, table, []string{"Ethernet2"}))
			after, err := test.GetData(ctx, vc, "|")
			require.NoError(t, err)
			require.Equal(t, before, after, "absent bindings must not change or create tables or fields")
		})
	}
}
