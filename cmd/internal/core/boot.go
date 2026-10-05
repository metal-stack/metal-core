package core

import (
	"fmt"
	"net/netip"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// validateRDNSS checks that the given RDNSS addresses are IPv6 addresses, the only ones FRR accepts in router advertisements.
func validateRDNSS(rdnss []string) error {
	for _, server := range rdnss {
		addr, err := netip.ParseAddr(server)
		if err != nil {
			return fmt.Errorf("invalid boot rdnss address %q: %w", server, err)
		}
		if !addr.Is6() || addr.Is4In6() {
			return fmt.Errorf("boot rdnss address %q must be an ipv6 address", server)
		}
	}
	return nil
}

// configureBoot adds the boot VRF and DNS servers to the switch configuration.
//
// The boot VNI and per port boot prefixes are assigned by the metal-apiserver from the partition's boot network;
// RDNSS addresses come from the partition's boot configuration.
func (c *Core) configureBoot(s *apiv2.Switch, cfg *types.Conf) error {
	if s.BootVni == nil {
		return fmt.Errorf("switch has no boot network assigned in partition %s", s.Partition)
	}
	if err := validateRDNSS(s.BootRdnss); err != nil {
		return err
	}

	cfg.Ports.Vrfs[types.BootVrfName] = &types.Vrf{VNI: *s.BootVni}
	cfg.BootRDNSS = s.BootRdnss
	return nil
}

// bootPrefix validates the API-assigned prefix. An invalid or missing prefix only
// prevents this port from being configured, rather than the whole switch.
func (c *Core) bootPrefix(nic *apiv2.SwitchNic) netip.Prefix {
	if nic.BootPrefix == nil {
		c.log.Warn("unprovisioned port has no boot prefix assigned by the metal-apiserver, port is left out", "port", nic.Name)
		return netip.Prefix{}
	}
	prefix, err := types.ParseBootPrefix(*nic.BootPrefix)
	if err != nil {
		c.log.Warn("unprovisioned port has an invalid boot prefix, port is left out", "port", nic.Name, "error", err)
		return netip.Prefix{}
	}
	return prefix
}
