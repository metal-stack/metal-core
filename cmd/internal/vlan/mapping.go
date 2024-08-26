package vlan

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

// Mapping holds the current mapping of VLAN IDs to VNIs of the switch
type Mapping map[uint16]uint32

// ReadMapping reads the current VLAN to VNI mapping with the help of netlink
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
