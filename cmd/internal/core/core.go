package core

import (
	"fmt"
	"log/slog"
	"net/netip"

	clientv2 "github.com/metal-stack/api/go/client"
	"github.com/metal-stack/metal-core/cmd/internal/metrics"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type (
	Core struct {
		log      *slog.Logger
		logLevel string

		cidr                    string
		loopbackIP              string
		asn                     string
		partitionID             string
		rackID                  string
		roomID                  string
		enableReconfigureSwitch bool
		managementGateway       string
		additionalMgmtRoutes    []string
		additionalBridgePorts   []string
		additionalBridgeVIDs    []string
		spineUplinks            []string
		pxeVlanID               uint16
		bgpNeighborStateFile    string
		setSrcLoopback          bool
		boot                    BootConfig

		nos     switcher.NOS
		client  clientv2.Client
		metrics *metrics.Metrics
	}

	Config struct {
		Log      *slog.Logger
		LogLevel string

		CIDR                  string
		LoopbackIP            string
		ASN                   string
		PartitionID           string
		RackID                string
		RoomID                string
		ReconfigureSwitch     bool
		ManagementGateway     string
		PXEVlanID             uint16
		BGPNeighborStateFile  string
		AdditionalMgmtRoutes  []string
		AdditionalBridgePorts []string
		AdditionalBridgeVIDs  []string
		SpineUplinks          []string
		SetSrcLoopback        bool
		Boot                  BootConfig

		NOS     switcher.NOS
		Client  clientv2.Client
		Metrics *metrics.Metrics
	}

	// BootMode defines how unprovisioned machines boot.
	BootMode string

	// BootConfig configures the boot of unprovisioned machines (MEP-20).
	BootConfig struct {
		// Mode is pxe (ports in the pxe vlan) or l3 (ports in the ipv6 boot vrf).
		Mode BootMode
		// VNI is the layer 3 vni of the boot vrf.
		VNI uint32
		// Prefix is the ipv6 prefix of this switch from which every unprovisioned port gets its own /64.
		// This is a temporary source until the metal-apiserver assigns the boot prefix per switch port.
		Prefix netip.Prefix
		// RDNSS are the dns server addresses advertised to booting machines.
		RDNSS []string
	}
)

const (
	// BootModePXE puts unprovisioned ports into the pxe vlan, machines boot via dhcp/pxe.
	BootModePXE = BootMode("pxe")
	// BootModeL3 puts unprovisioned ports into the ipv6 boot vrf, machines boot via slaac and a bmc mounted iso (MEP-20).
	BootModeL3 = BootMode("l3")
)

// NewBootConfig validates the given boot settings.
func NewBootConfig(mode string, vni uint32, prefix string, rdnss []string) (BootConfig, error) {
	c := BootConfig{
		Mode:  BootMode(mode),
		VNI:   vni,
		RDNSS: rdnss,
	}

	switch c.Mode {
	case BootModePXE:
		return c, nil
	case BootModeL3:
	default:
		return BootConfig{}, fmt.Errorf("unknown boot mode %q, must be one of %q or %q", mode, BootModePXE, BootModeL3)
	}

	if c.VNI == 0 || c.VNI > types.MaxVNI {
		return BootConfig{}, fmt.Errorf("boot mode %s requires a boot vni between 1 and %d", BootModeL3, types.MaxVNI)
	}

	p, err := netip.ParsePrefix(prefix)
	if err != nil {
		return BootConfig{}, fmt.Errorf("boot mode %s requires a valid ipv6 boot prefix: %w", BootModeL3, err)
	}
	if !p.Addr().Is6() || p.Addr().Is4In6() {
		return BootConfig{}, fmt.Errorf("boot prefix %s must be an ipv6 prefix", prefix)
	}
	if p.Bits() == 0 || p.Bits() > types.BootPrefixLength {
		return BootConfig{}, fmt.Errorf("boot prefix %s must be between /1 and /%d", prefix, types.BootPrefixLength)
	}
	c.Prefix = p.Masked()

	for _, server := range rdnss {
		addr, err := netip.ParseAddr(server)
		if err != nil {
			return BootConfig{}, fmt.Errorf("invalid boot rdnss address %q: %w", server, err)
		}
		if !addr.Is6() || addr.Is4In6() {
			return BootConfig{}, fmt.Errorf("boot rdnss address %q must be an ipv6 address", server)
		}
	}

	return c, nil
}

func New(c Config) *Core {
	return &Core{
		log:                     c.Log,
		logLevel:                c.LogLevel,
		cidr:                    c.CIDR,
		loopbackIP:              c.LoopbackIP,
		asn:                     c.ASN,
		partitionID:             c.PartitionID,
		rackID:                  c.RackID,
		roomID:                  c.RoomID,
		enableReconfigureSwitch: c.ReconfigureSwitch,
		managementGateway:       c.ManagementGateway,
		additionalMgmtRoutes:    c.AdditionalMgmtRoutes,
		additionalBridgePorts:   c.AdditionalBridgePorts,
		additionalBridgeVIDs:    c.AdditionalBridgeVIDs,
		spineUplinks:            c.SpineUplinks,
		setSrcLoopback:          c.SetSrcLoopback,
		nos:                     c.NOS,
		client:                  c.Client,
		metrics:                 c.Metrics,
		pxeVlanID:               c.PXEVlanID,
		bgpNeighborStateFile:    c.BGPNeighborStateFile,
		boot:                    c.Boot,
	}
}
