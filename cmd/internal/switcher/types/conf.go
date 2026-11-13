package types

import (
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
)

// FillVLANIDs fills the given configuration object with switch-local VLAN IDs
// if they are present in the given VLAN-Mapping
// otherwise: new available VLAN IDs will be used
func (c *Conf) FillVLANIDs(m vlan.Mapping) error {
outer_loop:
	for _, t := range c.Vrfs {
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

func (c *Conf) NewWithoutDownPorts() {
	var ports []Port

	for _, port := range c.Ports {
		if port.AdminStatus == PortStatusDown {
			continue
		}
		ports = append(ports, port)
	}

	c.Ports = ports
}
