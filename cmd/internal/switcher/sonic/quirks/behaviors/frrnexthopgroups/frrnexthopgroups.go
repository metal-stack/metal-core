package frrnexthopgroups

type FrrDisableNextHopGroups interface {
	// Do we need to disable NextHopGroups in FRR?
	// Seems like NextHopGroups are not supported in newer SONiC versions with dplane_fpm_sonic
	// (at least sonic-vpp 202605 and asternos 202511)
	Enabled() bool
}

type Pre202411 struct{}

func (Pre202411) Enabled() bool { return false }

type Post202411 struct{}

func (Post202411) Enabled() bool { return true }
