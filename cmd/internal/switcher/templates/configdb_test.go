package templates

import (
	"encoding/json"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVRFApplyer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			cfg := buildVrfConfig(&c, []string{"swp1", "swp1|10.0.0.10"})

			data, err := json.MarshalIndent(cfg, "", "  ")
			require.NoError(t, err, "Couldn't marshall VRF config")
			actual := strings.TrimSpace(string(data))
			expected := strings.TrimSpace(readExpected(t, path.Join("test_data", tt, "vrf_config.json")))
			require.Equal(t, expected, actual, "Wanted: %s, Got: %s", expected, actual)
		})
	}
}

func TestVLANApplyer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			cfg := buildVlanConfig(&c)

			data, err := json.MarshalIndent(cfg, "", "  ")
			require.NoError(t, err, "Couldn't marshall VLAN config")
			actual := string(data)
			expected := readExpected(t, path.Join("test_data", tt, "vlan_config.json"))
			require.Equal(t, expected, actual, "Wanted: %s, Got: %s", expected, actual)
		})
	}
}
