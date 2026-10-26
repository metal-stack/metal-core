package frrfpm

type FrrUseDplaneFpmSonic interface {
	// Is the SONiC distribution using the new dplane_fpm_sonic module to sync zebra with the Forwarding Plane Manager
	// https://github.com/sonic-net/SONiC/blob/master/doc/sonic-fpm-module/frr_sonic_communication_channel.md
	Enabled() bool
}

type Pre202411 struct{}

func (Pre202411) Enabled() bool { return false }

type Post202411 struct{}

func (Post202411) Enabled() bool { return true }
