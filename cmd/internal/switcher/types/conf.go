package types

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"slices"

	"github.com/metal-stack/metal-core/cmd/internal/vlan"
	"go4.org/netipx"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FillVLANIDs fills the given configuration object with switch-local VLAN IDs
// if they are present in the given VLAN-Mapping
// otherwise: new available VLAN IDs will be used
func (c *Conf) FillVLANIDs(m vlan.Mapping) error {
	for _, t := range c.Ports.Vrfs {
		vl, err := vlanIDForVNI(m, t.VNI)
		if err != nil {
			return err
		}
		t.VLANID = vl
	}
	if c.Boot != nil {
		for name, t := range c.Ports.Vrfs {
			if t.VNI == c.Boot.VNI {
				return fmt.Errorf("boot vni %d collides with the vni of vrf %s", c.Boot.VNI, name)
			}
		}
		vl, err := vlanIDForVNI(m, c.Boot.VNI)
		if err != nil {
			return err
		}
		c.Boot.VLANID = vl
	}
	return nil
}

func vlanIDForVNI(m vlan.Mapping, vni uint32) (uint16, error) {
	for vl, mappedVni := range m {
		if mappedVni == vni {
			return vl, nil
		}
	}
	vlanids, err := m.ReserveVlanIDs(1)
	if err != nil {
		return 0, err
	}
	vl := vlanids[0]
	m[vl] = vni
	return vl, nil
}

// DeriveBootPorts derives the per port boot prefix for every given port from the boot prefix block of the switch.
//
// The block is split into /64 prefixes, the port with ordinal n gets the n-th /64.
// The switch itself takes the first address of every prefix.
// This is a temporary derivation until the metal-apiserver assigns the boot prefix per switch port (MEP-20).
func DeriveBootPorts(block netip.Prefix, ordinals map[string]int, ports []string) (map[string]BootPort, error) {
	if !block.IsValid() || !block.Addr().Is6() || block.Addr().Is4In6() {
		return nil, fmt.Errorf("boot prefix %q must be an ipv6 prefix", block)
	}
	if block.Bits() == 0 || block.Bits() > BootPrefixLength {
		return nil, fmt.Errorf("boot prefix %q must be between /1 and /%d", block, BootPrefixLength)
	}
	block = block.Masked()

	var (
		available = uint64(1) << (BootPrefixLength - block.Bits())
		base      = block.Addr().As16()
		hi        = binary.BigEndian.Uint64(base[:8])
		result    = map[string]BootPort{}
	)

	for _, port := range ports {
		ordinal, ok := ordinals[port]
		if !ok {
			return nil, fmt.Errorf("no port ordinal known for port %q", port)
		}
		if ordinal < 0 || uint64(ordinal) >= available {
			return nil, fmt.Errorf("port ordinal %d of port %q exceeds the %d prefixes available in boot prefix %s", ordinal, port, available, block)
		}

		var addr [16]byte
		binary.BigEndian.PutUint64(addr[:8], hi|uint64(ordinal)) // nolint:gosec
		prefix := netip.PrefixFrom(netip.AddrFrom16(addr), BootPrefixLength)

		addr[15] = 1
		address := netip.PrefixFrom(netip.AddrFrom16(addr), BootPrefixLength)

		result[port] = BootPort{
			Prefix:  prefix.String(),
			Address: address.String(),
		}
	}

	return result, nil
}

func (c *Conf) FillRouteMapsAndIPPrefixLists() error {
	for port, f := range c.Ports.Firewalls {
		f.Assemble("fw-"+port, f.Vnis, f.Cidrs)
	}
	for vrf, t := range c.Ports.Vrfs {
		var err error
		t.Cidrs, err = compactCidrs(t.Cidrs)
		if err != nil {
			return err
		}

		cidrsByAf := cidrsByAddressfamily(t.Cidrs)
		t.Has4 = len(cidrsByAf.ipv4Cidrs) > 0
		t.Has6 = len(cidrsByAf.ipv6Cidrs) > 0
		t.Assemble(vrf, []string{}, t.Cidrs)
	}
	return nil
}
func compactCidrs(cidrs []string) ([]string, error) {
	var (
		compacted    []string
		ipsetBuilder netipx.IPSetBuilder
	)

	for _, cidr := range cidrs {
		parsed, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, err
		}
		ipsetBuilder.AddPrefix(parsed)
	}
	set, err := ipsetBuilder.IPSet()
	if err != nil {
		return nil, fmt.Errorf("unable to create ipset:%w", err)
	}
	for _, pfx := range set.Prefixes() {
		compacted = append(compacted, pfx.String())
	}

	return compacted, nil
}

// CapitalizeVrfName capitalizes VRF names, which is requirement for SONiC
func (c *Conf) CapitalizeVrfName() {
	caser := cases.Title(language.English)
	capitalizedVRFs := make(map[string]*Vrf)
	for name, vrf := range c.Ports.Vrfs {
		s := caser.String(name)
		capitalizedVRFs[s] = vrf
	}

	c.Ports.Vrfs = capitalizedVRFs
}

func (c *Conf) NewWithoutDownPorts() *Conf {
	var (
		underlay      []string
		unprovisioned []string
		bladePorts    []string
		vrfs          = map[string]*Vrf{}
		firewalls     = map[string]*Firewall{}
	)

	ports := c.Ports
	for _, port := range ports.Underlay {
		if ports.AdminStatus[port] != PortStatusDown {
			underlay = append(underlay, port)
		}
	}

	for _, port := range ports.Unprovisioned {
		if ports.AdminStatus[port] != PortStatusDown {
			unprovisioned = append(unprovisioned, port)
		}
	}

	for _, port := range ports.BladePorts {
		if ports.AdminStatus[port] != PortStatusDown {
			bladePorts = append(bladePorts, port)
		}
	}

	for name, vrf := range ports.Vrfs {
		newVrf := Vrf{
			Filter: vrf.Filter,
			VNI:    vrf.VNI,
			VLANID: vrf.VLANID,
			Cidrs:  vrf.Cidrs,
			Has4:   vrf.Has4,
			Has6:   vrf.Has6,
		}

		var neighbors []string
		for _, neigh := range vrf.Neighbors {
			if ports.AdminStatus[neigh] != PortStatusDown {
				neighbors = append(neighbors, neigh)
			}
		}
		newVrf.Neighbors = neighbors
		vrfs[name] = &newVrf
	}

	for name, fw := range ports.Firewalls {
		if ports.AdminStatus[fw.Port] == PortStatusDown {
			continue
		}
		firewalls[name] = &Firewall{
			Filter: fw.Filter,
			Port:   fw.Port,
			Cidrs:  fw.Cidrs,
			Vnis:   fw.Vnis,
		}
	}

	newConf := *c
	if c.Boot != nil {
		boot := *c.Boot
		boot.Ports = map[string]BootPort{}
		for port, bp := range c.Boot.Ports {
			if slices.Contains(unprovisioned, port) {
				boot.Ports[port] = bp
			}
		}
		newConf.Boot = &boot
	}
	newConf.Ports = Ports{
		Eth0:          c.Ports.Eth0,
		Underlay:      underlay,
		Unprovisioned: unprovisioned,
		BladePorts:    bladePorts,
		Vrfs:          vrfs,
		Firewalls:     firewalls,
		AdminStatus:   c.Ports.AdminStatus,
	}
	return &newConf
}
