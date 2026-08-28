package sonic

import (
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"google.golang.org/protobuf/testing/protocmp"
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
		status apiv2.SwitchPortStatus
		want   *apiv2.SwitchNic
	}{
		{
			name:   "naming schema empty",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: "",
			status: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP,
			want: &apiv2.SwitchNic{
				Name:       "Ethernet0",
				Identifier: "Eth1/1",
				State: &apiv2.NicState{
					Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UP,
				},
			},
		},
		{
			name:   "default",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaDefault,
			status: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN,
			want: &apiv2.SwitchNic{
				Name:       "Ethernet0",
				Identifier: "Eth1/1",
				State: &apiv2.NicState{
					Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN,
				},
			},
		},
		{
			name:   "swap",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaSwap,
			status: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UNKNOWN,
			want: &apiv2.SwitchNic{
				Name:       "Eth1/1",
				Identifier: "Ethernet0",
				State: &apiv2.NicState{
					Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_UNKNOWN,
				},
			},
		},
		{
			name:   "name",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaName,
			status: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN,
			want: &apiv2.SwitchNic{
				Name:       "Ethernet0",
				Identifier: "Ethernet0",
				State: &apiv2.NicState{
					Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN,
				},
			},
		},
		{
			name:   "alias",
			ifname: "Ethernet0",
			alias:  "Eth1/1",
			naming: InterfaceNamingSchemaAlias,
			status: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN,
			want: &apiv2.SwitchNic{
				Name:       "Eth1/1",
				Identifier: "Eth1/1",
				State: &apiv2.NicState{
					Actual: apiv2.SwitchPortStatus_SWITCH_PORT_STATUS_DOWN,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSwitchNicByNamingSchema(tt.ifname, tt.alias, tt.naming, tt.status)
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("getNicByNamingSchema() diff = %s", diff)
			}
		})
	}
}
