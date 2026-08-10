package sonic

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-go/api/models"
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

func Test_getSwitchNicByNamingSchema(t *testing.T) {
	tests := []struct {
		name   string
		ifname string
		alias  string
		naming InterfaceNamingSchema
		want   *models.V1SwitchNic
	}{
		{
			name:   "naming schema empty",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: "",
			want: &models.V1SwitchNic{
				Name:       new("Ethernet0"),
				Identifier: new("Eth1/1"),
			},
		},
		{
			name:   "default",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaDefault,
			want: &models.V1SwitchNic{
				Name:       new("Ethernet0"),
				Identifier: new("Eth1/1"),
			},
		},
		{
			name:   "swap",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaSwap,
			want: &models.V1SwitchNic{
				Name:       new("Eth1/1"),
				Identifier: new("Ethernet0"),
			},
		},
		{
			name:   "name",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaName,
			want: &models.V1SwitchNic{
				Name:       new("Ethernet0"),
				Identifier: new("Ethernet0"),
			},
		},
		{
			name:   "alias",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaAlias,
			want: &models.V1SwitchNic{
				Name:       new("Eth1/1"),
				Identifier: new("Eth1/1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSwitchNicByNamingSchema(tt.ifname, tt.alias, tt.naming)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("getNicByNamingSchema() diff = %s", diff)
			}
		})
	}
}
