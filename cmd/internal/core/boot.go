package core

import (
	"fmt"
	"net/netip"

	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type (
	// BootMode defines how unprovisioned machines boot.
	BootMode string

	// BootConfig configures the boot of unprovisioned machines.
	BootConfig struct {
		// Mode is "pxe" (ports in the PXE VLAN) or "l3" (ports in the IPv6 boot VRF).
		Mode BootMode
	}
)

const (
	// BootModePXE puts unprovisioned ports into the PXE VLAN; machines boot via DHCP/PXE.
	BootModePXE = BootMode("pxe")
	// BootModeL3 puts unprovisioned ports into the IPv6 boot VRF; machines boot via SLAAC and an ISO mounted through the BMC.
	BootModeL3 = BootMode("l3")
)

// NewBootConfig validates the given boot settings.
//
// In L3 boot mode, the boot VNI, per port boot prefixes, and RDNSS addresses are configured in the metal-apiserver
// and delivered with the switch (Switch.boot_vni, SwitchNic.boot_prefix, Switch.boot_rdnss).
func NewBootConfig(mode string) (BootConfig, error) {
	c := BootConfig{
		Mode: BootMode(mode),
	}

	switch c.Mode {
	case BootModePXE, BootModeL3:
		return c, nil
	default:
		return BootConfig{}, fmt.Errorf("unknown boot mode %q, must be one of %q or %q", mode, BootModePXE, BootModeL3)
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

// buildBootConfig builds the boot VRF configuration for the unprovisioned ports.
//
// The boot VNI and per port boot prefixes are assigned by the metal-apiserver from the partition's boot network;
// RDNSS addresses come from the partition's boot configuration.
// Ports without a valid boot prefix are left out, so that only they fail to be configured instead of the whole switch.
func (c *Core) buildBootConfig(s *apiv2.Switch, unprovisioned []string, bootPrefixes map[string]string) (*types.BootConf, error) {
	if s.BootVni == nil {
		return nil, fmt.Errorf("boot mode l3 requires a boot network in partition %s, the switch has no boot vni", s.Partition)
	}
	if err := validateRDNSS(s.BootRdnss); err != nil {
		return nil, err
	}

	ports := map[string]types.BootPort{}
	for _, port := range unprovisioned {
		prefix, ok := bootPrefixes[port]
		if !ok {
			c.log.Warn("unprovisioned port has no boot prefix assigned by the metal-apiserver, port is left out", "port", port)
			continue
		}
		bootPort, err := types.BootPortFromPrefix(prefix)
		if err != nil {
			c.log.Warn("unprovisioned port has an invalid boot prefix, port is left out", "port", port, "error", err)
			continue
		}
		ports[port] = bootPort
	}

	return &types.BootConf{
		Vrf:   types.BootVrfName,
		VNI:   *s.BootVni,
		RDNSS: s.BootRdnss,
		Ports: ports,
	}, nil
}
