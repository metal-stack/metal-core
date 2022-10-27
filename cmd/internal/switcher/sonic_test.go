package switcher

import (
	"github.com/stretchr/testify/require"
	"path"
	"testing"
)

func TestSonicGetPortsConfig(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			ports, err := getPortsConfig(path.Join("test_data", tt, "portsdb.json"))
			require.NoError(t, err, "Failed to get ports config")

			interfaceToAliasMap := map[string]string{
				"Ethernet0":  "fortyGigE0/0",
				"Ethernet4":  "fortyGigE1/0",
				"Ethernet8":  "fortyGigE2/0",
				"Ethernet12": "fortyGigE3/0",
			}
			require.Equal(
				t, len(interfaceToAliasMap), len(ports),
				"Expected ports config length: %d, Got: %d", len(interfaceToAliasMap), len(ports))

			for i, a := range interfaceToAliasMap {
				v, _ := ports[i]
				require.Equal(t, a, v.Alias, "Expected interface alias: %s, Got: %s", a, v.Alias)
			}
		})
	}
}
