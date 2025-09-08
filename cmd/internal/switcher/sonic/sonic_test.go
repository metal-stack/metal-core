package sonic

import (
	"log"
	"net"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func Test_getPortsConfig(t *testing.T) {
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
				v := ports[i]
				require.Equal(t, a, v.Alias, "Expected interface alias: %s, Got: %s", a, v.Alias)
			}
		})
	}
}

func listTestCases() []string {
	files, err := os.ReadDir("test_data")
	if err != nil {
		log.Fatal(err)
	}

	r := []string{}
	for _, f := range files {
		if f.IsDir() {
			r = append(r, f.Name())
		}
	}
	return r
}

func Test_portsToInterfaces(t *testing.T) {
	tests := []struct {
		name  string
		ports map[string]PortInfo
		want  []*net.Interface
	}{
		{
			name: "add port to slice of interfaces",
			ports: map[string]PortInfo{
				"Ethernet1": {
					Alias: "Eth1",
				},
			},
			want: []*net.Interface{
				{
					Name: "Ethernet1",
				},
			},
		},
		{
			name: "sort interfaces alphabetically",
			ports: map[string]PortInfo{
				"Ethernet1":  {},
				"Ethernet2":  {},
				"Ethernet10": {},
				"Ethernet3":  {},
				"Ethernet30": {},
			},
			want: []*net.Interface{
				{
					Name: "Ethernet1",
				},
				{
					Name: "Ethernet10",
				},
				{
					Name: "Ethernet2",
				},
				{
					Name: "Ethernet3",
				},
				{
					Name: "Ethernet30",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := portsToInterfaces(tt.ports)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("portsToInterfaces() diff = %s", diff)
			}
		})
	}
}
