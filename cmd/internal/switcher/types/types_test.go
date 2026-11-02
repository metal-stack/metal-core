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

// func TestPortHasAddressFamilies(t *testing.T) {
// 	tests := []struct {
// 		name  string
// 		cidrs []string
// 		want4 bool
// 		want6 bool
// 	}{
// 		{
// 			name:  "none",
// 			cidrs: []string{},
// 		},
// 		{
// 			name:  "only ipv6",
// 			cidrs: []string{"2001:db8:2::/48"},
// 			want6: true,
// 		},
// 		{
// 			name:  "only ipv4",
// 			cidrs: []string{"10.120.0.0/24"},
// 			want4: true,
// 		},
// 		{
// 			name:  "both",
// 			cidrs: []string{"10.120.0.0/24", "2001:db8:2::/48"},
// 			want4: true,
// 			want6: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			p := &Port{
// 				Cidrs: tt.cidrs,
// 			}
// 			got4, got6, err := p.HasAddressFamilies()
// 			if err != nil {
// 				t.Errorf("Port.HasAddressFamilies() error = %v", err)
// 			}
// 			if got4 != tt.want4 {
// 				t.Errorf("Port.HasAddressFamilies() ipv4 = %v, want %v", got4, tt.want4)
// 			}
// 			if got6 != tt.want6 {
// 				t.Errorf("Port.HasAddressFamilies() ipv6 = %v, want %v", got6, tt.want6)
// 			}
// 		})
// 	}
// }
