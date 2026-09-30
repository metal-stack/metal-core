package vlan

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

type Mapping map[uint16]uint32

func ReadMapping() (Mapping, error) {
	m := Mapping{}
	links, _ := netlink.LinkList()
	bvl, _ := netlink.BridgeVlanList()
	for _, b := range bvl {
		for _, e := range b {
			m[e.Vid] = 0
		}
	}
	for _, nic := range links {
		if nic.Type() == "vxlan" {
			vx := nic.(*netlink.Vxlan)
			vni := vx.VxlanId
			ifindex := int32(nic.Attrs().Index) // nolint:gosec
			if len(bvl[ifindex]) < 1 {
				return nil, fmt.Errorf("no vlan mapping could be determined for vxlan interface %s", nic.Attrs().Name)
			}
			vlan := bvl[ifindex][0].Vid
			m[vlan] = uint32(vni) // nolint:gosec
		}
	}
	return m, nil
}
