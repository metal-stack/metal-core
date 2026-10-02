package db

import (
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

var (
	stateDBTestData = test.StringMap{
		"INTERFACE_TABLE": test.StringMap{
			"Ethernet124": test.StringMap{
				"mac_addr": "74:fe:48:7c:27:38",
				"vrf":      "",
			},
			"Vlan4000": test.StringMap{
				"vrf": "",
			},
			"Vlan4000|10.255.3.1/24": test.StringMap{
				"state": "ok",
			},
		},
		"VLAN_MEMBER_TABLE": test.StringMap{
			"Vlan4000|Ethernet0": test.StringMap{
				"state": "ok",
			},
			"Vlan4000|Ethernet12": test.StringMap{
				"state": "ok",
			},
		},
	}
)

func TestStateDB_ExistInterface(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
		want          bool
	}{
		{
			name:          "routed port",
			interfaceName: "Ethernet124",
			want:          true,
		},
		{
			name:          "bridged port",
			interfaceName: "Ethernet0",
			want:          false,
		},
		{
			name:          "unknown port",
			interfaceName: "Ethernet120",
			want:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = "|"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, stateDBTestData, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &StateDB{
				c: c,
			}
			got, err := d.ExistInterface(ctx, tt.interfaceName)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("StateDB.ExistInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateDB_ExistVlanMember(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
		want          bool
	}{
		{
			name:          "bridged port",
			interfaceName: "Ethernet0",
			want:          true,
		},
		{
			name:          "port name is a prefix of a bridged port",
			interfaceName: "Ethernet1",
			want:          false,
		},
		{
			name:          "routed port",
			interfaceName: "Ethernet124",
			want:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctx = t.Context()
				sep = "|"
				vc  = test.StartValkey(t)
			)
			defer vc.Close()

			err := test.LoadData(ctx, vc, stateDBTestData, sep)
			require.NoError(t, err)

			c := &Client{
				rdb: vc,
				sep: sep,
			}
			d := &StateDB{
				c: c,
			}
			got, err := d.ExistVlanMember(ctx, tt.interfaceName)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("StateDB.ExistVlanMember() = %v, want %v", got, tt.want)
			}
		})
	}
}
