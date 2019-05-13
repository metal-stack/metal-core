package vlan

import "fmt"

// vlanIDMin specifies the min VLAN-ID we want to use on our switches
const vlanIDMin uint16 = 1001

// vlanIDMax specifies the max VLAN-ID we want to use on our switches
const vlanIDMax uint16 = 2000

// ReserveVlanIDs tries to reserve n VLAN-IDs given the current switch configuration
func (m Mapping) ReserveVlanIDs(n uint16) ([]uint16, error) {
	return m.reserveVlanIDs(vlanIDMin, vlanIDMax, n)
}

func (m Mapping) reserveVlanIDs(min, max, n uint16) ([]uint16, error) {
	maxVlans := max - min + 1
	if uint16(len(m))+n > maxVlans {
		return nil, fmt.Errorf("can not reserve %d vlan ids, %d are already taken and %d possible at max", n, len(m), maxVlans)
	}
	// scan vlan id range for n free vlan ids
	r := []uint16{}
	for i := min; i <= max; i++ {
		if uint16(len(r)) >= n {
			break
		}
		if _, has := m[i]; !has {
			r = append(r, i)
		}
	}
	return r, nil
}
