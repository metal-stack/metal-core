package redis

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func TestApplier_NeedsFrrFirst(t *testing.T) {
	// Ethernet0 is a machine port, Ethernet1 a firewall port, Ethernet2 is not routed
	// and Ethernet3 carries an ip address next to its interface configuration.
	data := test.StringMap{
		"INTERFACE": test.StringMap{
			"Ethernet0": test.StringMap{
				"ipv6_use_link_local_only": "enable",
				"vrf_name":                 "Vrf102",
			},
			"Ethernet1": test.StringMap{
				"ipv6_use_link_local_only": "enable",
			},
			"Ethernet3": test.StringMap{
				"ipv6_use_link_local_only": "enable",
			},
			"Ethernet3|10.0.0.1/24": test.StringMap{},
		},
	}

	tests := []struct {
		name          string
		unprovisioned []string
		want          bool
	}{
		{
			name:          "nothing is deprovisioned",
			unprovisioned: nil,
			want:          false,
		},
		{
			name:          "a firewall port is deprovisioned",
			unprovisioned: []string{"Ethernet1"},
			want:          true,
		},
		{
			name:          "a machine port is deprovisioned",
			unprovisioned: []string{"Ethernet0"},
			want:          false,
		},
		{
			name:          "a port that is not routed stays unprovisioned",
			unprovisioned: []string{"Ethernet2"},
			want:          false,
		},
		{
			name:          "a firewall port among machine ports is deprovisioned",
			unprovisioned: []string{"Ethernet0", "Ethernet2", "Ethernet1"},
			want:          true,
		},
		{
			name:          "a firewall port with an ip address is deprovisioned",
			unprovisioned: []string{"Ethernet3"},
			want:          true,
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

			err := test.LoadData(ctx, vc, data, sep)
			require.NoError(t, err)

			a := NewApplier(slog.New(slog.DiscardHandler), &db.DB{
				Config: db.NewConfigDB(vc, sep),
			})

			got, err := a.NeedsFrrFirst(ctx, &types.Conf{
				Ports: types.Ports{
					Unprovisioned: tt.unprovisioned,
				},
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
