package db

import (
	"net/netip"
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

func TestApplDB_GetRoutes(t *testing.T) {
	data := test.StringMap{
		"ROUTE_TABLE": test.StringMap{
			"Vrf20": test.StringMap{
				"10.128.0.1": test.StringMap{
					"ifname":  "Ethernet1",
					"nexthop": "fe80::527c:6fff:fe3e:8d90",
				},
				"2001:db8::/64": test.StringMap{
					"ifname":  "Ethernet6,Ethernet12",
					"nexthop": "fe80::1,fe80::2",
				},
			},
			"Vrf30": test.StringMap{
				"10.130.0.0/24": test.StringMap{
					"nexthop_group": "7",
				},
			},
			"10.0.0.0/24": test.StringMap{
				"ifname":  "Ethernet120",
				"nexthop": "fe80::3",
				// AsterNOS writes an empty next hop group into every route
				"nexthop_group": "",
			},
		},
		"NEXTHOP_GROUP_TABLE": test.StringMap{
			"7": test.StringMap{
				"ifname":  "Ethernet2,Ethernet3",
				"nexthop": "10.4.0.1,10.4.0.2",
			},
		},
	}
	tests := []struct {
		name string
		vrf  string
		want []Route
	}{
		{
			name: "routes of a vrf",
			vrf:  "Vrf20",
			want: []Route{
				{
					Key:      "ROUTE_TABLE:Vrf20:10.128.0.1",
					Vrf:      "Vrf20",
					Prefix:   netip.MustParsePrefix("10.128.0.1/32"),
					NextHops: []NextHop{{Gateway: "fe80::527c:6fff:fe3e:8d90", Interface: "Ethernet1"}},
				},
				{
					Key:    "ROUTE_TABLE:Vrf20:2001:db8::/64",
					Vrf:    "Vrf20",
					Prefix: netip.MustParsePrefix("2001:db8::/64"),
					NextHops: []NextHop{
						{Gateway: "fe80::1", Interface: "Ethernet6"},
						{Gateway: "fe80::2", Interface: "Ethernet12"},
					},
				},
			},
		},
		{
			name: "next hops of a next hop group",
			vrf:  "Vrf30",
			want: []Route{
				{
					Key:    "ROUTE_TABLE:Vrf30:10.130.0.0/24",
					Vrf:    "Vrf30",
					Prefix: netip.MustParsePrefix("10.130.0.0/24"),
					NextHops: []NextHop{
						{Gateway: "10.4.0.1", Interface: "Ethernet2"},
						{Gateway: "10.4.0.2", Interface: "Ethernet3"},
					},
				},
			},
		},
		{
			name: "vrf without routes",
			vrf:  "Vrf40",
			want: []Route{},
		},
		{
			name: "all vrfs",
			want: []Route{
				{
					Key:      "ROUTE_TABLE:10.0.0.0/24",
					Prefix:   netip.MustParsePrefix("10.0.0.0/24"),
					NextHops: []NextHop{{Gateway: "fe80::3", Interface: "Ethernet120"}},
				},
				{
					Key:      "ROUTE_TABLE:Vrf20:10.128.0.1",
					Vrf:      "Vrf20",
					Prefix:   netip.MustParsePrefix("10.128.0.1/32"),
					NextHops: []NextHop{{Gateway: "fe80::527c:6fff:fe3e:8d90", Interface: "Ethernet1"}},
				},
				{
					Key:    "ROUTE_TABLE:Vrf20:2001:db8::/64",
					Vrf:    "Vrf20",
					Prefix: netip.MustParsePrefix("2001:db8::/64"),
					NextHops: []NextHop{
						{Gateway: "fe80::1", Interface: "Ethernet6"},
						{Gateway: "fe80::2", Interface: "Ethernet12"},
					},
				},
				{
					Key:    "ROUTE_TABLE:Vrf30:10.130.0.0/24",
					Vrf:    "Vrf30",
					Prefix: netip.MustParsePrefix("10.130.0.0/24"),
					NextHops: []NextHop{
						{Gateway: "10.4.0.1", Interface: "Ethernet2"},
						{Gateway: "10.4.0.2", Interface: "Ethernet3"},
					},
				},
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

			err := test.LoadData(ctx, vc, data, sep)
			require.NoError(t, err)

			d := &ApplDB{
				c: &Client{
					rdb: vc,
					sep: sep,
				},
			}
			got, err := d.GetRoutes(ctx, tt.vrf)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestApplDB_GetRoutesMissingNextHopGroup(t *testing.T) {
	var (
		ctx = t.Context()
		sep = ":"
		vc  = test.StartValkey(t)
	)
	defer vc.Close()

	err := test.LoadData(ctx, vc, test.StringMap{
		"ROUTE_TABLE": test.StringMap{
			"Vrf20": test.StringMap{
				"10.128.0.0/24": test.StringMap{
					"nexthop_group": "8",
				},
			},
		},
	}, sep)
	require.NoError(t, err)

	d := &ApplDB{
		c: &Client{
			rdb: vc,
			sep: sep,
		},
	}
	_, err = d.GetRoutes(ctx, "Vrf20")
	require.ErrorContains(t, err, "next hop group 8 of route ROUTE_TABLE:Vrf20:10.128.0.0/24 does not exist")
}
