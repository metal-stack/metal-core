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

func (c *Conf) NewWithoutDownPorts() *Conf {
	// var (
	// 	underlay      []string
	// 	unprovisioned []string
	// 	bladePorts    []string
	// 	vrfs          = map[string]*Vrf{}
	// 	firewalls     = map[string]*Firewall{}
	// )

	// ports := c.Ports
	// for _, port := range c.UnderlayPorts {
	// 	if ports.AdminStatus[port] != PortStatusDown {
	// 		underlay = append(underlay, port)
	// 	}
	// }

	// for _, port := range ports.Unprovisioned {
	// 	if ports.AdminStatus[port] != PortStatusDown {
	// 		unprovisioned = append(unprovisioned, port)
	// 	}
	// }

	// for _, port := range ports.BladePorts {
	// 	if ports.AdminStatus[port] != PortStatusDown {
	// 		bladePorts = append(bladePorts, port)
	// 	}
	// }

	// for name, vrf := range ports.Vrfs {
	// 	newVrf := Vrf{
	// 		Filter: vrf.Filter,
	// 		VNI:    vrf.VNI,
	// 		VLANID: vrf.VLANID,
	// 		Cidrs:  vrf.Cidrs,
	// 		Has4:   vrf.Has4,
	// 		Has6:   vrf.Has6,
	// 	}

	// 	var neighbors []string
	// 	for _, neigh := range vrf.Neighbors {
	// 		if ports.AdminStatus[neigh] != PortStatusDown {
	// 			neighbors = append(neighbors, neigh)
	// 		}
	// 	}
	// 	newVrf.Neighbors = neighbors
	// 	vrfs[name] = &newVrf
	// }

	// for name, fw := range ports.Firewalls {
	// 	if ports.AdminStatus[fw.Port] == PortStatusDown {
	// 		continue
	// 	}
	// 	firewalls[name] = &Firewall{
	// 		Filter: fw.Filter,
	// 		Port:   fw.Port,
	// 		Cidrs:  fw.Cidrs,
	// 		Vnis:   fw.Vnis,
	// 	}
	// }

	newConf := *c
	// newConf.Ports = Ports{
	// 	Eth0:          c.Ports.Eth0,
	// 	UnderlayPorts: underlay,
	// 	Unprovisioned: unprovisioned,
	// 	BladePorts:    bladePorts,
	// 	Vrfs:          vrfs,
	// 	Firewalls:     firewalls,
	// 	AdminStatus:   c.Ports.AdminStatus,
	// }
	return &newConf
}
