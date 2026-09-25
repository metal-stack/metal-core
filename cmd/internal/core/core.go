package core

import (
	"fmt"
	"log/slog"
	"net/netip"

	clientv2 "github.com/metal-stack/api/go/client"
	"github.com/metal-stack/metal-core/cmd/internal/metrics"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
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
//
// In boot mode l3 the boot vni and the per port boot prefixes are assigned by the metal-apiserver
// and delivered with the switch (Switch.boot_vni, SwitchNic.boot_prefix).
func NewBootConfig(mode string, rdnss []string) (BootConfig, error) {
	c := BootConfig{
		Mode:  BootMode(mode),
		RDNSS: rdnss,
	}

	switch c.Mode {
	case BootModePXE:
		return c, nil
	case BootModeL3:
	default:
		return BootConfig{}, fmt.Errorf("unknown boot mode %q, must be one of %q or %q", mode, BootModePXE, BootModeL3)
	}

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
