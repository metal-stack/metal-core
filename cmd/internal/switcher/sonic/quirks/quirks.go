package quirks

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

var QuirkProfiles = map[string]func() Quirks{
	"community": func() Quirks { return communityQuirks{} },
}

// Quirks describes what differs between SONiC distributions.
// Quirks only return data, the applier decides how and when to apply it.
type Quirks interface {
	// Name of the quirks profile
	Name() string
	// VrfEntries returns the CONFIG_DB entries that make up a VRF in the order they need to be created.
	// They are deleted in reverse order.
	VrfEntries(vrfName string, vrf *types.Vrf, vtep string) []db.Entry
	// PortState returns how to tell whether a port is still bridged or routed
	PortState(log *slog.Logger, sonicDb *db.DB) PortStateChecker
}

// communityQuirks is upstream SONiC behaviour. Profiles that only differ in some
// aspects can embed it and override only what differs.
type communityQuirks struct{}

func (communityQuirks) Name() string { return "community" }

// Returns the objects that are required to create an EVPN VXLAN according to
// https://github.com/sonic-net/SONiC/blob/master/doc/vxlan/EVPN/EVPN_VXLAN_HLD.md#511-configuration-commands
func (communityQuirks) VrfEntries(vrfName string, vrf *types.Vrf, vtep string) []db.Entry {
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

func (communityQuirks) PortState(log *slog.Logger, sonicDb *db.DB) PortStateChecker {
	return &asicDBPortState{db: sonicDb, log: log}
}
