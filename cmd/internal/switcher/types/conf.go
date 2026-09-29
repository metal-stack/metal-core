package types

import (
	"fmt"
	"net/netip"

	"github.com/metal-stack/metal-core/cmd/internal/vlan"
	"go4.org/netipx"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FillVLANIDs fills the given configuration object with switch-local VLAN IDs
// if they are present in the given VLAN-Mapping
// otherwise: new available VLAN IDs will be used
func (c *Conf) FillVLANIDs(m vlan.Mapping) error {
	if boot := c.Ports.Vrfs[BootVrfName]; boot != nil {
		for name, vrf := range c.Ports.Vrfs {
			if name != BootVrfName && vrf.VNI == boot.VNI {
				return fmt.Errorf("boot vni %d collides with the vni of vrf %s", boot.VNI, name)
			}
		}
	}
	for _, vrf := range c.Ports.Vrfs {
		vl, err := vlanIDForVNI(m, vrf.VNI)
		if err != nil {
			return err
		}
		vrf.VLANID = vl
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

// ParseBootPrefix validates and masks a boot prefix assigned by the metal-apiserver.
// The prefix must be an IPv6 /64.
func ParseBootPrefix(prefix string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(prefix)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("invalid boot prefix %q: %w", prefix, err)
	}
	if !p.Addr().Is6() || p.Addr().Is4In6() {
		return netip.Prefix{}, fmt.Errorf("boot prefix %q must be an ipv6 prefix", prefix)
	}
	if p.Bits() != BootPrefixLength {
		return netip.Prefix{}, fmt.Errorf("boot prefix %q must be a /%d", prefix, BootPrefixLength)
	}
	return p.Masked(), nil
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
		if name != BootVrfName {
			name = caser.String(name)
		}
		capitalizedVRFs[name] = vrf
	}

	c.Ports.Vrfs = capitalizedVRFs
}

func (c *Conf) NewWithoutDownPorts() *Conf {
	var (
		underlay      []string
		unprovisioned = map[string]*UnprovisionedPort{}
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

	for name, port := range ports.Unprovisioned {
		if ports.AdminStatus[port.Port] != PortStatusDown {
			unprovisioned[name] = &UnprovisionedPort{
				Port:       port.Port,
				BootPrefix: port.BootPrefix,
			}
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
