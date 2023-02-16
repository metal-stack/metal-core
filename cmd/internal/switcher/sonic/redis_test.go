package sonic

import (
	"testing"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/configdb"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/stretchr/testify/require"
)

type dummyConfigDB struct {
	Ifaces []configdb.InterfaceConfiguration
}

func (d *dummyConfigDB) ConfigureInterface(c configdb.InterfaceConfiguration) error {
	d.Ifaces = append(d.Ifaces, c)
	return nil
}

func TestApply(t *testing.T) {
	mock := &dummyConfigDB{}
	testee := &redisApplier{c: mock}

	err := testee.apply(&types.Conf{
		Ports: types.Ports{
			Unprovisioned: []string{"Ethernet0", "Ethernet1"},
			Vrfs: map[string]*types.Vrf{
				"Vrf20": {Neighbors: []string{"Ethernet2", "Ethernet4"}},
			},
		},
	})

	require.Nil(t, err)

	require.Equal(t, mock.Ifaces[0].Name, "Ethernet0")
	require.Equal(t, mock.Ifaces[0].Vlan.Name, "Vlan4000")
	require.Equal(t, mock.Ifaces[1].Name, "Ethernet1")
	require.Equal(t, mock.Ifaces[1].Vlan.Name, "Vlan4000")

	require.Equal(t, mock.Ifaces[2].Name, "Ethernet2")
	require.Equal(t, mock.Ifaces[2].Vrf.Name, "Vrf20")
	require.Equal(t, mock.Ifaces[3].Name, "Ethernet4")
	require.Equal(t, mock.Ifaces[3].Vrf.Name, "Vrf20")
}