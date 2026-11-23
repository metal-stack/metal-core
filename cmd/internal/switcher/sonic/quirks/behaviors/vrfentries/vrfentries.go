package vrfentries

import (
	"strconv"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/object"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type VrfEntries interface {
	// Objects returns the CONFIG_DB entries that make up a VRF. Their order does not matter, it follows from
	// what each of them requires.
	Objects(vrfName string, vrf *types.Vrf, vtep string) []object.Object
}

type Community struct{}

// Objects returns the objects that are required to create an EVPN VXLAN according to
// https://github.com/sonic-net/SONiC/blob/master/doc/vxlan/EVPN/EVPN_VXLAN_HLD.md#511-configuration-commands
func (Community) Objects(vrfName string, vrf *types.Vrf, vtep string) []object.Object {
	vlan := object.VlanName(vrf.VLANID)

	return []object.Object{
		object.NewVrf(vrfName, db.Val{"fallback": "false", "vni": strconv.FormatUint(uint64(vrf.VNI), 10)}),
		object.NewVlan(vrf.VLANID),
		object.NewSuppressVlanNeigh(vlan),
		object.NewVlanInterface(vlan, vrfName),
		object.NewVxlanTunnelMap(vtep, vlan, vrf.VNI),
	}
}

// L3Only uses an L3-only VNI for platforms that do not support mapping the VRF VNI to a VLAN (e.g. AsterNOS),
// so the VRF is bound to the VTEP directly and no VLAN, SVI or tunnel map is created.
// ARP suppression is skipped, as it is an L2 protocol.
type L3Only struct{}

func (L3Only) Objects(vrfName string, vrf *types.Vrf, vtep string) []object.Object {
	return []object.Object{
		object.NewVrf(vrfName, db.Val{"vni": strconv.FormatUint(uint64(vrf.VNI), 10), "source_vtep": vtep}),
	}
}
