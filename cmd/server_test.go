//go:build client

package cmd

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func Test_validateBGPNeighbors(t *testing.T) {
	tests := []struct {
		name    string
		conf    *types.FRRConfig
		wantErr error
	}{
		{
			name: "unnumbered interfaces must be contained in unnumbered interfaces",
			conf: &types.FRRConfig{
				BGPs: []*types.BGP{
					{
						Neighbors: []types.BGPNeighbor{{Name: "Ethernet0 interface"}},
					},
				},
			},
			wantErr: fmt.Errorf(`unnumbered neighbor "Ethernet0" is missing from unnumbered interfaces`),
		},
		{
			name: "unnumbered neighbor missing `interface` suffix",
			conf: &types.FRRConfig{
				UnnumberedInterfaces: []*types.Interface{{Name: "Ethernet0"}},
				BGPs: []*types.BGP{
					{
						Neighbors: []types.BGPNeighbor{{Name: "Ethernet0"}},
					},
				},
			},
			wantErr: fmt.Errorf(`neighbor "Ethernet0" is not a valid ip address; if you are trying to add an unnumbered neighbor add the suffix "interface": "Ethernet0 interface"`),
		},
		{
			name: "referencing non-existent peer-group",
			conf: &types.FRRConfig{
				UnnumberedInterfaces: []*types.Interface{{Name: "Ethernet0"}},
				BGPs: []*types.BGP{
					{
						Neighbors: []types.BGPNeighbor{
							{
								Name:      "Ethernet0 interface",
								PeerGroup: new("FABRIC"),
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf(`peer-group "FABRIC" for neighbor "Ethernet0 interface" does not exist`),
		},
		{
			name: "unnumbered interface is bound to different vrf than its bgp session",
			conf: &types.FRRConfig{
				UnnumberedInterfaces: []*types.Interface{
					{
						Name: "Ethernet0",
						Vrf:  new("Vrf100"),
					},
				},
				BGPs: []*types.BGP{
					{
						Neighbors: []types.BGPNeighbor{
							{
								Name: "Ethernet0 interface",
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf(`unnumbered neighbor "Ethernet0" is bound to vrf "Vrf100" but is added as neighbor in vrf ""`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBGPNeighbors(tt.conf)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("validateBGPNeighbors() diff = %s", diff)
			}
		})
	}
}

func Test_validateUnnumberedInterfaces(t *testing.T) {
	tests := []struct {
		name    string
		conf    *types.FRRConfig
		nics    []*apiv2.SwitchNic
		wantErr error
	}{
		{
			name: "referenced vrf is missing",
			conf: &types.FRRConfig{
				UnnumberedInterfaces: []*types.Interface{
					{
						Name: "Ethernet0",
						Vrf:  new("Vrf100"),
					},
				},
			},
			nics:    []*apiv2.SwitchNic{{Name: "Ethernet0"}},
			wantErr: fmt.Errorf(`unnumbered interface "Ethernet0" is bound to vrf "Vrf100" which is missing from vrf config`),
		},
		{
			name: "empty vrf is allowed",
			conf: &types.FRRConfig{
				UnnumberedInterfaces: []*types.Interface{{Name: "Ethernet0"}},
			},
			nics:    []*apiv2.SwitchNic{{Name: "Ethernet0"}},
			wantErr: nil,
		},
		{
			name: "referenced nic is missing",
			conf: &types.FRRConfig{
				UnnumberedInterfaces: []*types.Interface{{Name: "Ethernet0"}},
			},
			wantErr: fmt.Errorf(`unnumbered interface "Ethernet0" is missing from switch nics`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUnnumberedInterfaces(tt.conf, tt.nics)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("validateUnnumberedInterfaces() diff = %s", diff)
			}
		})
	}
}

func Test_validateAddressFamiliesCommands(t *testing.T) {
	tests := []struct {
		name    string
		conf    *types.FRRConfig
		wantErr error
	}{
		{
			name: "route-map from global commands in ipv4 unicast missing",
			conf: &types.FRRConfig{
				BGPs: []*types.BGP{
					{
						AddressFamilies: types.BGPAddressFamilies{
							IPv4Unicast: &types.BGPAddressFamily{
								GlobalCommands: []string{"redistribute connected route-map DENY_MGMT"},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf(`global commands for address-family "ipv4 unicast" in vrf "" failed: %w`, fmt.Errorf(`command "redistribute connected route-map DENY_MGMT" references route-map "DENY_MGMT" which is not configured`)),
		},
		{
			name: "route-map from neighbor commands in ipv4 unicast missing",
			conf: &types.FRRConfig{
				BGPs: []*types.BGP{
					{
						Vrf: new("Vrf100"),
						AddressFamilies: types.BGPAddressFamilies{
							IPv4Unicast: &types.BGPAddressFamily{
								Neighbors: []*types.AddressFamilyNeighbor{
									{
										Name:     "MACHINE",
										Commands: []string{"route-map Vrf100-in in"},
									},
								},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf(`neighbor commands for neighbor "MACHINE" for address-family "ipv4 unicast" in vrf "Vrf100" failed: %w`, fmt.Errorf(`command "route-map Vrf100-in in" references route-map "Vrf100-in" which is not configured`)),
		},
		{
			name: "route-map from global commands in ipv6 unicast missing",
			conf: &types.FRRConfig{
				BGPs: []*types.BGP{
					{
						AddressFamilies: types.BGPAddressFamilies{
							IPv6Unicast: &types.BGPAddressFamily{
								GlobalCommands: []string{"redistribute connected route-map DENY_MGMT"},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf(`global commands for address-family "ipv6 unicast" in vrf "" failed: %w`, fmt.Errorf(`command "redistribute connected route-map DENY_MGMT" references route-map "DENY_MGMT" which is not configured`)),
		},
		{
			name: "route-map from neighbor commands in ipv6 unicast missing",
			conf: &types.FRRConfig{
				BGPs: []*types.BGP{
					{
						AddressFamilies: types.BGPAddressFamilies{
							IPv6Unicast: &types.BGPAddressFamily{
								Neighbors: []*types.AddressFamilyNeighbor{
									{
										Name:     "MACHINE",
										Commands: []string{"route-map Vrf100-in in"},
									},
								},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf(`neighbor commands for neighbor "MACHINE" for address-family "ipv6 unicast" in vrf "" failed: %w`, fmt.Errorf(`command "route-map Vrf100-in in" references route-map "Vrf100-in" which is not configured`)),
		},
		{
			name: "route-map from global commands in ipv6 unicast missing",
			conf: &types.FRRConfig{
				BGPs: []*types.BGP{
					{
						AddressFamilies: types.BGPAddressFamilies{
							L2VPNEVPN: &types.BGPAddressFamily{
								GlobalCommands: []string{"redistribute connected route-map DENY_MGMT"},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf(`global commands for address-family "l2vpn evpn" in vrf "" failed: %w`, fmt.Errorf(`command "redistribute connected route-map DENY_MGMT" references route-map "DENY_MGMT" which is not configured`)),
		},
		{
			name: "route-map from neighbor commands in l2vpn evpn missing",
			conf: &types.FRRConfig{
				BGPs: []*types.BGP{
					{
						AddressFamilies: types.BGPAddressFamilies{
							L2VPNEVPN: &types.BGPAddressFamily{
								Neighbors: []*types.AddressFamilyNeighbor{
									{
										Name:     "MACHINE",
										Commands: []string{"route-map Vrf100-in in"},
									},
								},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf(`neighbor commands for neighbor "MACHINE" for address-family "l2vpn evpn" in vrf "" failed: %w`, fmt.Errorf(`command "route-map Vrf100-in in" references route-map "Vrf100-in" which is not configured`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAddressFamiliesCommands(tt.conf)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("validateAddressFamiliesCommands() diff = %s", diff)
			}
		})
	}
}

func Test_validateCommandRouteMaps(t *testing.T) {
	tests := []struct {
		name    string
		cmds    []string
		rms     []*types.RouteMap
		wantErr error
	}{
		{
			name:    "route-map missing",
			cmds:    []string{"redistribute connected route-map DENY_MGMT"},
			wantErr: fmt.Errorf(`command "redistribute connected route-map DENY_MGMT" references route-map "DENY_MGMT" which is not configured`),
		},
		{
			name:    "route-map in missing",
			cmds:    []string{"route-map Vrf100-in in"},
			wantErr: fmt.Errorf(`command "route-map Vrf100-in in" references route-map "Vrf100-in" which is not configured`),
		},
		{
			name:    "route-map out missing",
			cmds:    []string{"route-map Vrf100-out out"},
			wantErr: fmt.Errorf(`command "route-map Vrf100-out out" references route-map "Vrf100-out" which is not configured`),
		},
		{
			name:    "route-map exists",
			cmds:    []string{"redistribute connected route-map DENY_MGMT"},
			rms:     []*types.RouteMap{{Name: "DENY_MGMT"}},
			wantErr: nil,
		},
		{
			name: "route-map in exists",
			cmds: []string{"route-map Vrf100-in in"},
			rms:  []*types.RouteMap{{Name: "Vrf100-in"}},
		},
		{
			name: "route-map out exists",
			cmds: []string{"route-map Vrf100-out out"},
			rms:  []*types.RouteMap{{Name: "Vrf100-out"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommandRouteMaps(tt.cmds, tt.rms)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("validateCommandRouteMaps() diff = %s", diff)
			}
		})
	}
}

func Test_validateAddressFamiliesNeighbors(t *testing.T) {
	tests := []struct {
		name    string
		bgp     *types.BGP
		wantErr error
	}{
		{
			name: "neighbor validation fails for ipv4 unicast",
			bgp: &types.BGP{
				Vrf: new("Vrf100"),
				AddressFamilies: types.BGPAddressFamilies{
					IPv4Unicast: &types.BGPAddressFamily{
						Neighbors: []*types.AddressFamilyNeighbor{{Name: "MACHINE"}},
					},
				},
			},
			wantErr: fmt.Errorf(`neighbor validation for address-family "ipv4 unicast" in vrf "Vrf100" failed: %w`, fmt.Errorf(`neighbor "MACHINE" is neither a peer-group nor a bgp neighbor`)),
		},
		{
			name: "neighbor validation fails for ipv6 unicast",
			bgp: &types.BGP{
				Vrf: new("Vrf100"),
				AddressFamilies: types.BGPAddressFamilies{
					IPv6Unicast: &types.BGPAddressFamily{
						Neighbors: []*types.AddressFamilyNeighbor{{Name: "MACHINE"}},
					},
				},
			},
			wantErr: fmt.Errorf(`neighbor validation for address-family "ipv6 unicast" in vrf "Vrf100" failed: %w`, fmt.Errorf(`neighbor "MACHINE" is neither a peer-group nor a bgp neighbor`)),
		},
		{
			name: "neighbor validation fails for l2vpn evpn",
			bgp: &types.BGP{
				Vrf: new("Vrf100"),
				AddressFamilies: types.BGPAddressFamilies{
					L2VPNEVPN: &types.BGPAddressFamily{
						Neighbors: []*types.AddressFamilyNeighbor{{Name: "MACHINE"}},
					},
				},
			},
			wantErr: fmt.Errorf(`neighbor validation for address-family "l2vpn evpn" in vrf "Vrf100" failed: %w`, fmt.Errorf(`neighbor "MACHINE" is neither a peer-group nor a bgp neighbor`)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAddressFamiliesNeighbors(tt.bgp)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("validateAddressFamiliesNeighbors() diff = %s", diff)
			}
		})
	}
}

func Test_validateAddressFamilyNeighbors(t *testing.T) {
	tests := []struct {
		name       string
		afNeighs   []*types.AddressFamilyNeighbor
		peerGroups []types.PeerGroup
		neighs     []types.BGPNeighbor
		wantErr    error
	}{
		{
			name:     "peer-group missing",
			afNeighs: []*types.AddressFamilyNeighbor{{Name: "MACHINE"}},
			wantErr:  fmt.Errorf(`neighbor "MACHINE" is neither a peer-group nor a bgp neighbor`),
		},
		{
			name:     "neighbor missing",
			afNeighs: []*types.AddressFamilyNeighbor{{Name: "Ethernet0"}},
			wantErr:  fmt.Errorf(`neighbor "Ethernet0" is neither a peer-group nor a bgp neighbor`),
		},
		{
			name:       "peer-group exists",
			afNeighs:   []*types.AddressFamilyNeighbor{{Name: "MACHINE"}},
			peerGroups: []types.PeerGroup{{Name: "MACHINE"}},
		},
		{
			name:     "unnumbered neighbor exists",
			afNeighs: []*types.AddressFamilyNeighbor{{Name: "Ethernet0"}},
			neighs:   []types.BGPNeighbor{{Name: "Ethernet0 interface"}},
		},
		{
			name:     "numbered neighbor exists",
			afNeighs: []*types.AddressFamilyNeighbor{{Name: "10.0.0.1"}},
			neighs:   []types.BGPNeighbor{{Name: "10.0.0.1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAddressFamilyNeighbors(tt.afNeighs, tt.peerGroups, tt.neighs)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("validateAddressFamilyNeighbors() diff = %s", diff)
			}
		})
	}
}

func Test_validateRouteMaps(t *testing.T) {
	tests := []struct {
		name    string
		conf    *types.FRRConfig
		wantErr error
	}{
		{
			name: "duplicate order not allowed",
			conf: &types.FRRConfig{
				RouteMaps: []*types.RouteMap{
					{
						Name:  "Vrf100-in",
						Order: 10,
					},
					{
						Name:  "Vrf100-in",
						Order: 10,
					},
				},
			},
			wantErr: fmt.Errorf(`route-map "Vrf100-in" is defined multiple times with the same order 10`),
		},
		{
			name: "policy must be deny or permit",
			conf: &types.FRRConfig{
				RouteMaps: []*types.RouteMap{
					{
						Name:   "Vrf100-in",
						Policy: "allow",
					},
				},
			},
			wantErr: fmt.Errorf(`route-map "Vrf100-in" has policy "allow", must be %q or %q`, types.PolicyDeny, types.PolicyPermit),
		},
		{
			name: "order <= 0 is not allowed",
			conf: &types.FRRConfig{
				RouteMaps: []*types.RouteMap{
					{
						Name:   "Vrf100-in",
						Policy: types.PolicyPermit,
						Order:  0,
					},
				},
			},
			wantErr: fmt.Errorf(`route-map "Vrf100-in" has order 0, choose an order > 0`),
		},
		{
			name: "route-map references missing prefix-list",
			conf: &types.FRRConfig{
				RouteMaps: []*types.RouteMap{
					{
						Name:    "Vrf100-in",
						Entries: []string{"match ip address prefix-list Vrf100-in-prefixes"},
						Policy:  types.PolicyPermit,
						Order:   10,
					},
				},
			},
			wantErr: fmt.Errorf(`route-map "Vrf100-in" references prefix-list "Vrf100-in-prefixes" which is not defined`),
		},
		{
			name: "validation passes",
			conf: &types.FRRConfig{
				RouteMaps: []*types.RouteMap{
					{
						Name:    "Vrf100-in",
						Entries: []string{"match ip address prefix-list Vrf100-in-prefixes"},
						Policy:  types.PolicyPermit,
						Order:   10,
					},
				},
				PrefixLists: []*types.IPPrefixList{{Name: "Vrf100-in-prefixes"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRouteMaps(tt.conf)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("validateRouteMaps() diff = %s", diff)
			}
		})
	}
}

func Test_validatePrefixLists(t *testing.T) {
	tests := []struct {
		name    string
		pfs     []*types.IPPrefixList
		wantErr error
	}{
		{
			name: "address family ipv6 with ipv4 prefixes",
			pfs: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv6,
					Name:          "Vrf100-in6-prefixes",
					Spec:          "permit 10.0.0.0/16 le 32",
				},
			},
			wantErr: fmt.Errorf(`prefix-list "Vrf100-in6-prefixes" is an ipv6 prefix-list but references an ipv4 cidr "10.0.0.0/16"`),
		},
		{
			name: "address family ipv4 with ipv6 prefixes",
			pfs: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit fe80:10e2::/64 ge 128",
				},
			},
			wantErr: fmt.Errorf(`prefix-list "Vrf100-in-prefixes" is an ipv4 prefix-list but references an ipv6 cidr "fe80:10e2::/64"`),
		},
		{
			name: "ipv4 size > 32",
			pfs: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv4,
					Name:          "Vrf100-in-prefixes",
					Spec:          "permit 10.0.0.0/16 le 33",
				},
			},
			wantErr: fmt.Errorf(`prefix-list "Vrf100-in-prefixes" uses a mask size of 33, where it must be 0 <= size <= 32`),
		},
		{
			name: "ipv6 size > 128",
			pfs: []*types.IPPrefixList{
				{
					AddressFamily: types.AddressFamilyIPv6,
					Name:          "Vrf100-in6-prefixes",
					Spec:          "permit fe80:10e2::/64 le 130",
				},
			},
			wantErr: fmt.Errorf(`prefix-list "Vrf100-in6-prefixes" uses a mask size of 130, where it must be 0 <= size <= 128`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePrefixLists(tt.pfs)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("validatePrefixLists() diff = %s", diff)
			}
		})
	}
}
