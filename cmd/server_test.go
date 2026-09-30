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
			name:     "disallow passing fields other than neighbors and cidrs",
			filePath: path.Join(t.TempDir(), "malformed.yaml"),
			data: map[string]any{
				"vrf100": map[string]any{
					"Filter": map[string]any{},
					"VNI":    map[string]any{},
					"VLANID": map[string]any{},
					"Has4":   false,
					"Has6":   false,
				},
			},
			want:    nil,
			wantErr: fmt.Errorf("failed to unmarshal static VRFs file: %w", errors.New("yaml: construct errors: line 2: field Filter not found in type types.Vrf; line 3: field Has4 not found in type types.Vrf; line 4: field Has6 not found in type types.Vrf; line 5: field VLANID not found in type types.Vrf; line 6: field VNI not found in type types.Vrf")),
		},
		{
			name:     "parse vrfs",
			filePath: path.Join(t.TempDir(), "static-vrfs.yaml"),
			data: map[string]any{
				"vrf100": map[string]any{
					"neighbors": []string{"Ethernet0"},
					"cidrs":     []string{"10.10.1.0/24"},
				},
				"vrf200": map[string]any{
					"neighbors": []string{"Ethernet1"},
					"cidrs":     []string{"10.10.2.0/24"},
				},
			},
			want: types.Vrfs{
				"vrf100": {
					VNI:       100,
					Neighbors: []string{"Ethernet0"},
					Cidrs:     []string{"10.10.1.0/24"},
				},
				"vrf200": {
					VNI:       200,
					Neighbors: []string{"Ethernet1"},
					Cidrs:     []string{"10.10.2.0/24"},
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
				t.Errorf("getStaticVRFs() error = %v, want error %v", err, tt.wantErr)
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
