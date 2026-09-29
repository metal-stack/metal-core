package core

import (
	"fmt"
	"net/netip"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

// BootMode defines how unprovisioned machines boot.
type BootMode string

const (
	// BootModePXE puts unprovisioned ports into the PXE VLAN; machines boot via DHCP/PXE.
	BootModePXE = BootMode("pxe")
	// BootModeL3 puts unprovisioned ports into the IPv6 boot VRF; machines boot via SLAAC and an ISO mounted through the BMC.
	BootModeL3 = BootMode("l3")
)

// ParseBootMode validates the configured boot mode.
func ParseBootMode(mode string) (BootMode, error) {
	switch BootMode(mode) {
	case BootModePXE, BootModeL3:
		return BootMode(mode), nil
	default:
		return "", fmt.Errorf("unknown boot mode %q, must be one of %q or %q", mode, BootModePXE, BootModeL3)
	}
}

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
		return fmt.Errorf("boot mode l3 requires a boot network in partition %s, the switch has no boot vni", s.Partition)
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
