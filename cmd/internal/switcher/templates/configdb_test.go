package templates

import (
	"encoding/json"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigdbRenderer(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			cfg := buildConfigdb(&c, []string{"swp1", "swp1|10.0.0.10"})

			data, err := json.MarshalIndent(cfg, "", "  ")
			require.NoError(t, err, "Couldn't marshall configdb")
			actual := string(data)
			expected := readExpected(t, path.Join("test_data", tt, "configdb.json"))
			require.Equal(t, expected, actual, "Wanted: %s, Got: %s", expected, actual)
		})
	}
}
