package db

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRouteKey(t *testing.T) {
	tests := []struct {
		key        string
		wantVrf    string
		wantPrefix string
		wantErr    bool
	}{
		{key: "ROUTE_TABLE:10.0.0.0/24", wantPrefix: "10.0.0.0/24"},
		{key: "ROUTE_TABLE:10.0.0.11", wantPrefix: "10.0.0.11/32"},
		{key: "ROUTE_TABLE:fe80::/64", wantPrefix: "fe80::/64"},
		{key: "ROUTE_TABLE:Vrf20:0.0.0.0/0", wantVrf: "Vrf20", wantPrefix: "0.0.0.0/0"},
		{key: "ROUTE_TABLE:Vrf46:fe80::/64", wantVrf: "Vrf46", wantPrefix: "fe80::/64"},
		{key: "ROUTE_TABLE:Vrf20:2001:db8::1", wantVrf: "Vrf20", wantPrefix: "2001:db8::1/128"},
		{key: "ROUTE_TABLE:Vrf20:no-prefix", wantErr: true},
		{key: "INTF_TABLE:Ethernet0", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			vrf, prefix, err := parseRouteKey(tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantVrf, vrf)
			require.Equal(t, netip.MustParsePrefix(tt.wantPrefix), prefix)
		})
	}
}
