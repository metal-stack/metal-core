package switcher

import (
	"bufio"
	"os"

	"git.f-i-ts.de/cloud-native/metallib/vlan"
)

const FrrTmp = "/etc/frr/frr.tmp"
const Frr = "/etc/frr/frr.conf"
const IfacesTmp = "/etc/network/interfaces.tmp"
const Ifaces = "/etc/network/interfaces"

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

func (c *Conf) apply(tmpFile *os.File, dest *os.File, a Applier) error {
	w := bufio.NewWriter(tmpFile)
	err := a.Render(w)
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}
	err = a.Validate(tmpFile.Name())
	if err != nil {
		return err
	}
	err = os.Rename(tmpFile.Name(), dest.Name())
	if err != nil {
		return err
	}
	err = a.Reload()
	if err != nil {
		return err
	}
	return nil
}

func (c *Conf) applyFrr() error {
	tmp, err := os.OpenFile(FrrTmp, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer tmp.Close()

	f, err := os.OpenFile(Frr, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	frr := NewFrrApplier(c)
	err = c.apply(tmp, f, frr)
	if err != nil {
		return err
	}
	return nil
}

func (c *Conf) applyInterfaces() error {
	tmp, err := os.OpenFile(IfacesTmp, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer tmp.Close()

	f, err := os.OpenFile(Ifaces, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	ifaces := NewInterfacesApplier(c)
	err = c.apply(tmp, f, ifaces)
	if err != nil {
		return err
	}
	return nil
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
