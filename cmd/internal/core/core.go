package core

import (
	"log/slog"

	clientv2 "github.com/metal-stack/api/go/client"
	apiv2 "github.com/metal-stack/api/go/metalstack/api/v2"
	"github.com/metal-stack/metal-core/cmd/internal/metrics"
	"github.com/metal-stack/metal-core/cmd/internal/net"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"github.com/metal-stack/metal-core/cmd/internal/vlan"
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
		bootMode                BootMode

		nos     switcher.NOS
		client  clientv2.Client
		metrics *metrics.Metrics

		fillManagementInfo func(*types.Conf, string) error
		linkStatus         func(string) (apiv2.SwitchPortStatus, error)
		vlanMapping        func() (vlan.Mapping, error)
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
		BootMode              BootMode

		NOS     switcher.NOS
		Client  clientv2.Client
		Metrics *metrics.Metrics
	}
)

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
		fillManagementInfo:      fillManagementInfo,
		linkStatus:              net.GetLinkStatus,
		vlanMapping:             vlan.ReadMapping,
		client:                  c.Client,
		metrics:                 c.Metrics,
		pxeVlanID:               c.PXEVlanID,
		bgpNeighborStateFile:    c.BGPNeighborStateFile,
		bootMode:                c.BootMode,
	}
}
