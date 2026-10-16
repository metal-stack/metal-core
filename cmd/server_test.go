//go:build client

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"go.yaml.in/yaml/v4"
)

func Test_getStaticBGPConfig(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		data     map[string]any
		want     *types.StaticBGPConfig
		wantErr  error
	}{
		{
			name:     "empty file path",
			filePath: "",
			want:     nil,
			wantErr:  nil,
		},
		{
			name:     "fail to parse malformed file",
			filePath: path.Join(t.TempDir(), "malformed.yaml"),
			data: map[string]any{
				"bgp_sessions": "invalid",
			},
			want:    nil,
			wantErr: fmt.Errorf("failed to unmarshal static bgp config file: %w", errors.New("yaml: construct errors: line 1: cannot construct !!str `invalid` into []types.BGPSession")),
		},
		{
			name:     "can't set unnumbered explicitly",
			filePath: path.Join(t.TempDir(), "malformed.yaml"),
			data: map[string]any{
				"bgp_sessions": []map[string]any{
					{
						"vrf": "default",
						"neighbors": map[string]any{
							"1.1.1.1": map[string]any{
								"peer_group": "FABRIC",
								"unnumbered": true,
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: fmt.Errorf("failed to unmarshal static bgp config file: %w", errors.New("yaml: construct errors: line 5: field unnumbered not found in type types.BGPPeer")),
		},
		{
			name:     "successfully parse",
			filePath: path.Join(t.TempDir(), "static-bgp-config.yaml"),
			data: map[string]any{
				"bgp_sessions": []map[string]any{
					{
						"vrf": "default",
						"neighbors": map[string]any{
							"Ethernet0": map[string]any{
								"peer_group": "FABRIC",
							},
						},
					},
					{
						"vrf": "vrf100",
						"neighbors": map[string]any{
							"1.1.1.1": map[string]any{
								"peer_group": "MULTIHOP",
							},
						},
					},
				},
				"peer_groups": map[string]map[string]any{
					"default": {
						"FABRIC": map[string]any{
							"cidrs": []string{"1.1.1.0/24"},
							"evpn":  true,
						},
					},
					"vrf100": {
						"MULTIHOP": map[string]any{
							"cidrs": []string{"2.2.2.0/24"},
							"filter": map[string]any{
								"ip_prefix_lists": []map[string]any{
									{
										"address_family": "ip",
										"name":           "vrf100-in-prefixes",
										"spec":           "permit 10.1.1.0/24 ge 32",
									},
								},
								"route_maps": []map[string]any{
									{
										"name":    "vrf100-in",
										"entries": []string{"match ip address prefix-list vrf100-in-prefixes"},
										"policy":  "permit",
										"order":   5,
									},
								},
							},
							"commands": []string{"ebgp-multihop 2"},
						},
					},
				},
			},
			want: &types.StaticBGPConfig{
				BGPSessions: []types.BGPSession{
					{
						VRF: "default",
						Neighbors: map[string]types.BGPPeer{
							"Ethernet0": {
								PeerGroup:  "FABRIC",
								Unnumbered: true,
							},
						},
					},
					{
						VRF: "vrf100",
						Neighbors: map[string]types.BGPPeer{
							"1.1.1.1": {
								PeerGroup: "MULTIHOP",
							},
						},
					},
				},
				PeerGroups: map[string]map[string]types.PeerGroup{
					"default": {
						"FABRIC": {
							Cidrs: []string{"1.1.1.0/24"},
							EVPN:  true,
						},
					},
					"vrf100": {
						"MULTIHOP": {
							Cidrs: []string{"2.2.2.0/24"},
							Filter: types.Filter{
								IPPrefixLists: []types.IPPrefixList{
									{
										AddressFamily: types.AddressFamilyIPv4,
										Name:          "vrf100-in-prefixes",
										Spec:          "permit 10.1.1.0/24 ge 32",
									},
								},
								RouteMaps: []types.RouteMap{
									{
										Name:    "vrf100-in",
										Entries: []string{"match ip address prefix-list vrf100-in-prefixes"},
										Policy:  types.RouteMapPolicyPermit,
										Order:   5,
									},
								},
							},
							Commands: []string{"ebgp-multihop 2"},
						},
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.filePath != "" {
				err := createTestFile(tt.filePath, tt.data)
				if err != nil {
					t.Errorf("failed to create test file %s: %v", tt.filePath, err)
					return
				}
			}
			got, err := getStaticBGPConfig(tt.filePath)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("getStaticBGPConfig() error = %v, want error %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getStaticBGPConfig() diff = %s", diff)
			}
		})
	}
}

func createTestFile(file string, content map[string]any) error {
	data, err := yaml.Marshal(content)
	if err != nil {
		return err
	}
	err = os.WriteFile(file, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
