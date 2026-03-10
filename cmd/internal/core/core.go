package core

import (
	"log/slog"

	clientv2 "github.com/metal-stack/api/go/client"
	"github.com/metal-stack/metal-core/cmd/internal/metrics"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
)

type Core struct {
	log      *slog.Logger
	logLevel string

	cidr                    string
	loopbackIP              string
	asn                     string
	partitionID             string
	rackID                  string
	enableReconfigureSwitch bool
	managementGateway       string
	additionalBridgePorts   []string
	additionalBridgeVIDs    []string
	spineUplinks            []string
	pxeVlanID               uint16
	bgpNeighborStateFile    string

	nos     switcher.NOS
	client  clientv2.Client
	metrics *metrics.Metrics
}

type Config struct {
	Log      *slog.Logger
	LogLevel string

	CIDR                  string
	LoopbackIP            string
	ASN                   string
	PartitionID           string
	RackID                string
	ReconfigureSwitch     bool
	ManagementGateway     string
	PXEVlanID             uint16
	BGPNeighborStateFile  string
	AdditionalBridgePorts []string
	AdditionalBridgeVIDs  []string
	SpineUplinks          []string

	NOS     switcher.NOS
	Client  clientv2.Client
	Metrics *metrics.Metrics
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
		enableReconfigureSwitch: c.ReconfigureSwitch,
		managementGateway:       c.ManagementGateway,
		additionalBridgePorts:   c.AdditionalBridgePorts,
		additionalBridgeVIDs:    c.AdditionalBridgeVIDs,
		spineUplinks:            c.SpineUplinks,
		nos:                     c.NOS,
		client:                  c.Client,
		metrics:                 c.Metrics,
		pxeVlanID:               c.PXEVlanID,
		bgpNeighborStateFile:    c.BGPNeighborStateFile,
	}
}
