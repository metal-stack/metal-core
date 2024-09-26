package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_cidrsByAddressfamily(t *testing.T) {
	tests := []struct {
		name  string
		cidrs []string
		want  cidrsByAf
	}{
		{
			name:  "simple",
			cidrs: []string{"10.0.0.0/8", "2001:db8:7::/48", "192.168.178.0/24", "2001:db8:2::/48"},
			want: cidrsByAf{
				ipv4Cidrs: []string{"10.0.0.0/8", "192.168.178.0/24"},
				ipv6Cidrs: []string{"2001:db8:7::/48", "2001:db8:2::/48"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cidrsByAddressfamily(tt.cidrs)
			require.ElementsMatch(t, got.ipv4Cidrs, tt.want.ipv4Cidrs)
			require.ElementsMatch(t, got.ipv6Cidrs, tt.want.ipv6Cidrs)
		})
	}
}
