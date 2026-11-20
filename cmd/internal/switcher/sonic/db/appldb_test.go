package db

import (
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

func TestApplDB_ExistPortInitDone(t *testing.T) {
	tests := []struct {
		name string
		data test.StringMap
		want bool
	}{
		{
			name: "not exists",
			data: test.StringMap{
				"PORT_TABLE": test.StringMap{
					"PortConfigDone": test.StringMap{
						"count": "65",
					},
				},
			},
			want: false,
		},
		{
			name: "exists",
			data: test.StringMap{
				"PORT_TABLE": test.StringMap{
					"PortInitDone": test.StringMap{
						"lanes": "0",
					},
				},
			},
			want: true,
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
			d := &ApplDB{
				c: c,
			}
			got, err := d.ExistPortInitDone(ctx)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("ApplDB.ExistPortInitDone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplDB_GetRoutesVia(t *testing.T) {
	data := test.StringMap{
		"ROUTE_TABLE": test.StringMap{
			"Vrf20": test.StringMap{
				"10.128.0.1": test.StringMap{
					"ifname": "Ethernet1",
				},
				"10.128.0.2": test.StringMap{
					"ifname":  "Ethernet1",
					"nexthop": "fe80::527c:6fff:fe3e:8d90",
				},
				"2001:db8::/64": test.StringMap{
					"ifname": "Ethernet6,Ethernet12",
				},
			},
			"10.0.0.0/24": test.StringMap{
				"ifname": "Ethernet120,Ethernet124",
			},
		},
	}
	tests := []struct {
		name          string
		vrf           string
		interfaceName string
		want          []string
	}{
		{
			name:          "routes via the interface",
			vrf:           "Vrf20",
			interfaceName: "Ethernet1",
			want:          []string{"ROUTE_TABLE:Vrf20:10.128.0.1", "ROUTE_TABLE:Vrf20:10.128.0.2"},
		},
		{
			name:          "next hop of an ecmp ipv6 route",
			vrf:           "Vrf20",
			interfaceName: "Ethernet12",
			want:          []string{"ROUTE_TABLE:Vrf20:2001:db8::/64"},
		},
		{
			name:          "route in another vrf",
			vrf:           "Vrf20",
			interfaceName: "Ethernet120",
		},
		{
			name:          "prefix of an interface name",
			vrf:           "Vrf20",
			interfaceName: "Ethernet",
		},
		{
			name:          "all vrfs",
			interfaceName: "Ethernet124",
			want:          []string{"ROUTE_TABLE:10.0.0.0/24"},
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

			err := test.LoadData(ctx, vc, data, sep)
			require.NoError(t, err)

			d := &ApplDB{
				c: &Client{
					rdb: vc,
					sep: sep,
				},
			}
			got, err := d.GetRoutesVia(ctx, tt.vrf, tt.interfaceName)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
