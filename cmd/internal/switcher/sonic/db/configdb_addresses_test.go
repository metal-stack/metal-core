package db

import (
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

func TestConfigDB_InterfaceAddresses(t *testing.T) {
	ctx := t.Context()
	vc := test.StartValkey(t)
	defer vc.Close()
	d := newConfigDB(vc, "|")
	require.NoError(t, test.LoadData(ctx, vc, test.StringMap{
		"INTERFACE": test.StringMap{
			"Ethernet0":                test.StringMap{"vrf_name": "Vrf102"},
			"Ethernet0|10.1.1.1/24":    test.StringMap{},
			"Ethernet0|2001:db8::1/64": test.StringMap{},
			"Ethernet1":                test.StringMap{"vrf_name": "Vrf102"},
			"Ethernet10|10.1.1.1/24":   test.StringMap{},
		},
	}, "|"))
	before, err := test.GetData(ctx, vc, "|")
	require.NoError(t, err)
	addresses, err := d.GetInterfaceAddresses(ctx, "Ethernet0")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"10.1.1.1/24", "2001:db8::1/64"}, addresses)
	for _, name := range []string{"Ethernet1", "EthernetMissing"} {
		addresses, err = d.GetInterfaceAddresses(ctx, name)
		require.NoError(t, err)
		require.Empty(t, addresses)
	}
	for _, address := range []string{"192.0.2.1/24", "2001:db8:1::1/64"} {
		require.NoError(t, d.SetInterfaceAddress(ctx, "Ethernet0", address))
		require.NoError(t, d.SetInterfaceAddress(ctx, "Ethernet0", address))
		all, err := test.GetData(ctx, vc, "|")
		require.NoError(t, err)
		require.Equal(t, map[string]string{"NULL": "NULL"}, all["INTERFACE|Ethernet0|"+address])
		delete(all, "INTERFACE|Ethernet0|"+address)
		require.Equal(t, before, all)
		require.NoError(t, d.DeleteInterfaceAddress(ctx, "Ethernet0", address))
		require.NoError(t, d.DeleteInterfaceAddress(ctx, "Ethernet0", address))
		all, err = test.GetData(ctx, vc, "|")
		require.NoError(t, err)
		require.Equal(t, before, all)
	}
	require.NoError(t, d.DeleteInterfaceAddress(ctx, "Ethernet0", "10.1.1.1/24"))
	delete(before, "INTERFACE|Ethernet0|10.1.1.1/24")
	all, err := test.GetData(ctx, vc, "|")
	require.NoError(t, err)
	require.Equal(t, before, all)
}
