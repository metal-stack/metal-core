package sonic

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
)

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

func Test_getNicByNamingSchema(t *testing.T) {
	tests := []struct {
		name   string
		ifname string
		alias  string
		naming InterfaceNamingSchema
		want   db.Port
	}{
		{
			name:   "naming schema empty",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: "",
			want: db.Port{
				Name:  "Ethernet0",
				Alias: "Eth1/1",
			},
		},
		{
			name:   "default",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaDefault,
			want: db.Port{
				Name:  "Ethernet0",
				Alias: "Eth1/1",
			},
		},
		{
			name:   "swap",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaSwap,
			want: db.Port{
				Name:  "Eth1/1",
				Alias: "Ethernet0",
			},
		},
		{
			name:   "name",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaName,
			want: db.Port{
				Name:  "Ethernet0",
				Alias: "Ethernet0",
			},
		},
		{
			name:   "alias",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaAlias,
			want: db.Port{
				Name:  "Eth1/1",
				Alias: "Eth1/1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPortByNamingSchema(tt.ifname, tt.alias, tt.naming)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getNicByNamingSchema() diff = %s", diff)
			}
		})
	}
}
