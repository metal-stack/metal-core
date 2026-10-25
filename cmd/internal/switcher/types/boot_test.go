package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapitalizeVrfNamePreservesBootName(t *testing.T) {
	boot := &Vrf{VNI: 104000}
	tenant := &Vrf{VNI: 104001}
	c := &Conf{Ports: Ports{Vrfs: Vrfs{BootVrfName: boot, "vrf104001": tenant}}}
	c.CapitalizeVrfName()
	require.Equal(t, Vrfs{"VrfBoot": boot, "Vrf104001": tenant}, c.Ports.Vrfs)
}
