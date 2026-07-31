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

func Test_getStaticVRFs(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		data     map[string]any
		want     types.Vrfs
		wantErr  error
	}{
		{
			name:     "empty file path",
			filePath: "",
			want:     types.Vrfs{},
			wantErr:  nil,
		},
		{
			name:     "fail to parse malformed file",
			filePath: path.Join(t.TempDir(), "malformed.yaml"),
			data: map[string]any{
				"vrf100": "invalid",
			},
			want:    nil,
			wantErr: fmt.Errorf("failed to unmarshal static VRFs file: %w", errors.New("yaml: construct errors: line 1: cannot construct !!str `invalid` into types.Vrf")),
		},
		{
			name:     "parse vrfs",
			filePath: path.Join(t.TempDir(), "static-vrfs.yaml"),
			data: map[string]any{
				"vrf100": map[string]any{
					"filter": map[string]any{
						"ip-prefix-lists": []map[string]any{
							{
								"address-family": "ip",
								"name":           "vrf100-in-prefixes",
								"spec":           "permit 10.10.0.0/24 le 32",
							},
						},
						"route-maps": []map[string]any{
							{
								"name":    "vrf100-in",
								"entries": []string{"match ip address prefix-list vrf100-in-prefixes"},
								"policy":  "permit",
								"order":   10,
							},
						},
					},
					"vni":       100,
					"vlanid":    1000,
					"neighbors": []string{"Ethernet0"},
					"cidrs":     []string{"10.10.1.0/24"},
					"has4":      true,
					"has6":      false,
				},
				"vrf200": map[string]any{
					"vni":       200,
					"vlanid":    2000,
					"neighbors": []string{"Ethernet1"},
					"cidrs":     []string{"10.10.2.0/24"},
					"has4":      true,
					"has6":      true,
				},
			},
			want: types.Vrfs{
				"vrf100": {
					Filter: types.Filter{
						IPPrefixLists: []types.IPPrefixList{
							{
								AddressFamily: "ip",
								Name:          "vrf100-in-prefixes",
								Spec:          "permit 10.10.0.0/24 le 32",
							},
						},
						RouteMaps: []types.RouteMap{
							{
								Name:    "vrf100-in",
								Entries: []string{"match ip address prefix-list vrf100-in-prefixes"},
								Policy:  "permit",
								Order:   10,
							},
						},
					},
					VNI:       100,
					VLANID:    1000,
					Neighbors: []string{"Ethernet0"},
					Cidrs:     []string{"10.10.1.0/24"},
					Has4:      true,
					Has6:      false,
				},
				"vrf200": {
					VNI:       200,
					VLANID:    2000,
					Neighbors: []string{"Ethernet1"},
					Cidrs:     []string{"10.10.2.0/24"},
					Has4:      true,
					Has6:      true,
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
			got, err := getStaticVRFs(tt.filePath)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("getStaticVRFs() error diff = %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getStaticVRFs() diff = %s", diff)
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
