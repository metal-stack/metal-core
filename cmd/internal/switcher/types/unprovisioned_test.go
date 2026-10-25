package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewWithoutDownPortsCopiesUnprovisionedPorts(t *testing.T) {
	c := &Conf{Ports: Ports{
		Unprovisioned: map[string]*UnprovisionedPort{
			"swp1": {Port: "swp1"},
			"swp2": {Port: "swp2"},
		},
		AdminStatus: map[string]PortStatus{"swp2": PortStatusDown},
	}}
	got := c.NewWithoutDownPorts()
	require.Equal(t, map[string]*UnprovisionedPort{"swp1": {Port: "swp1"}}, got.Ports.Unprovisioned)
	require.NotSame(t, c.Ports.Unprovisioned["swp1"], got.Ports.Unprovisioned["swp1"])
	require.Len(t, c.Ports.Unprovisioned, 2)
}
