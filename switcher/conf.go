package switcher

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/vlan"
)

// FillVLANIDs fills the given configuration object with switch-local VLAN-IDs
// if they are present in the given VLAN-Mapping
// otherwise: new available VLAN-IDs will be used
func (c *Conf) FillVLANIDs(m vlan.Mapping) error {
Tloop:
	for _, t := range c.Tenants {
		for vlan, vni := range m {
			if vni == t.VNI {
				t.VLANID = vlan
				continue Tloop
			}
		}
		vlanids, err := m.ReserveVlanIDs(1)
		if err != nil {
			return err
		}
		vlan := vlanids[0]
		t.VLANID = vlan
		m[vlan] = t.VNI
	}
	return nil
}

func (c *Conf) applyFrr() error {
	a := NewFrrApplier(c)
	return a.Apply()
}

func (c *Conf) applyInterfaces() error {
	a := NewInterfacesApplier(c)
	return a.Apply()
}

// Apply applies the configuration to the switch
func (c *Conf) Apply() error {
	err := c.applyInterfaces()
	if err != nil {
		return err
	}

	err = c.applyFrr()
	if err != nil {
		return err
	}
	return nil
}
