package switcher

import (
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FillVLANIDs fills the given configuration object with switch-local VLAN-IDs
// if they are present in the given VLAN-Mapping
// otherwise: new available VLAN-IDs will be used
func (c *Conf) FillVLANIDs(m vlan.Mapping) error {
outer_loop:
	for _, t := range c.Ports.Vrfs {
		for vl, vni := range m {
			if vni == t.VNI {
				t.VLANID = vl
				continue outer_loop
			}
		}
		vlanids, err := m.ReserveVlanIDs(1)
		if err != nil {
			return err
		}
		vl := vlanids[0]
		t.VLANID = vl
		m[vl] = t.VNI
	}
	return nil
}

func (c *Conf) FillRouteMapsAndIPPrefixLists() {
	for port, f := range c.Ports.Firewalls {
		f.Assemble("fw-"+port, f.Vnis, f.Cidrs)
	}
	for vrf, t := range c.Ports.Vrfs {
		podCidr := "10.244.0.0/16"
		t.Cidrs = append(t.Cidrs, podCidr)
		t.Assemble(vrf, []string{}, t.Cidrs)
	}
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
	return
}
