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
	"asternos":  func() Quirks { return asterNOSQuirks{} },
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
	// Is the SONiC distribution using the new dplane_fpm_sonic module to sync zebra with the Forwarding Plane Manager
	// https://github.com/sonic-net/SONiC/blob/master/doc/sonic-fpm-module/frr_sonic_communication_channel.md
	frrUseDplaneFpmSonic() bool
	// Do we need to disable NextHopGroups in FRR?
	// Seems like NextHopGroups are not supported in newer SONiC versions with dplane_fpm_sonic
	// (at least sonic-vpp 202605 and asternos 202511)
	frrDisableNextHopGroups() bool
	// Remove Interface configuration instead of deleting them in FRR
	// Required true on newer FRR versions (observed on asternos 202511)
	frrKeepInterfaces() bool
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

func (communityQuirks) frrUseDplaneFpmSonic() bool { return false }

func (communityQuirks) frrDisableNextHopGroups() bool { return false }

func (communityQuirks) frrKeepInterfaces() bool { return false }

//nolint:unused
type asterNOSQuirks struct{ communityQuirks }

func (asterNOSQuirks) Name() string { return "asternos" }

// VrfEntries uses an L3-only VNI: AsterNOS does not support mapping the VRF VNI to a VLAN,
// so the VRF is bound to the VTEP directly and no VLAN, SVI or tunnel map is created.
// ARP suppression is skipped, as it is an L2 protocol
func (asterNOSQuirks) VrfEntries(vrfName string, vrf *types.Vrf, vtep string) []db.Entry {
	return []db.Entry{
		{Key: db.Key{"VRF", vrfName}, Val: db.Val{"vni": strconv.FormatUint(uint64(vrf.VNI), 10), "source_vtep": vtep}},
	}
}

// PortState checks the STATE_DB instead of the ASIC_DB.
func (asterNOSQuirks) PortState(log *slog.Logger, sonicDb *db.DB) PortStateChecker {
	return &stateDBPortState{db: sonicDb}
}

// Asternos uses dplane_fpm_sonic
func (asterNOSQuirks) frrUseDplaneFpmSonic() bool { return true }

// FRR 10.3 in Asternos refuses to apply a frr.conf that deletes active interfaces
func (asterNOSQuirks) frrKeepInterfaces() bool { return true }

// Asternos does not support NextHopGroups with dplane_fpm_sonic
func (asterNOSQuirks) frrDisableNextHopGroups() bool { return true }

// Apply the Quirks to the FRR Configuration
//
// Fun Fact: Go does not have virtual dispatch in member methods. Making this a
// member of the interface and defining the community implementation only would
// make the asternos quirks call the q.frrKeepInterfaces of the community
// implementation even if it is overridden
func ApplyFRRTemplateQuirks(q Quirks, cfg types.Conf) *types.Conf {
	cfg.KeepUnprovisionedInterfaces = q.frrKeepInterfaces()
	cfg.DisableNextHopGroups = q.frrDisableNextHopGroups()
	cfg.SetFPMAddress = q.frrUseDplaneFpmSonic()
	return &cfg
}
