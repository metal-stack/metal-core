package vrfentries

import (
	"fmt"
	"strconv"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type VrfEntries interface {
	// Entries returns the CONFIG_DB entries that make up a VRF in the order they need to be created.
	// They are deleted in reverse order.
	Entries(vrfName string, vrf *types.Vrf, vtep string) []db.Entry
}

type Community struct{}

// Entries returns the objects that are required to create an EVPN VXLAN according to
// https://github.com/sonic-net/SONiC/blob/master/doc/vxlan/EVPN/EVPN_VXLAN_HLD.md#511-configuration-commands
func (Community) Entries(vrfName string, vrf *types.Vrf, vtep string) []db.Entry {
	vlan := fmt.Sprintf("Vlan%d", vrf.VLANID)
	vni := strconv.FormatUint(uint64(vrf.VNI), 10)

	return []db.Entry{
		{Key: db.Key{"VRF", vrfName}, Val: db.Val{"fallback": "false", "vni": vni}},
		{Key: db.Key{"VLAN", vlan}, Val: db.Val{"vlanid": strconv.FormatUint(uint64(vrf.VLANID), 10)}},
		{Key: db.Key{"SUPPRESS_VLAN_NEIGH", vlan}, Val: db.Val{"suppress": "on"}},
		{Key: db.Key{"VLAN_INTERFACE", vlan}, Val: db.Val{"vrf_name": vrfName}},
		{Key: db.Key{"VXLAN_TUNNEL_MAP", vtep, "map_" + vni + "_" + vlan}, Val: db.Val{"vlan": vlan, "vni": vni}},
	}
}

// L3Only uses an L3-only VNI for platforms that do not support mapping the VRF VNI to a VLAN (e.g. AsterNOS),
// so the VRF is bound to the VTEP directly and no VLAN, SVI or tunnel map is created.
// ARP suppression is skipped, as it is an L2 protocol.
type L3Only struct{}

func (L3Only) Entries(vrfName string, vrf *types.Vrf, vtep string) []db.Entry {
	return []db.Entry{
		{Key: db.Key{"VRF", vrfName}, Val: db.Val{"vni": strconv.FormatUint(uint64(vrf.VNI), 10), "source_vtep": vtep}},
	}
}
