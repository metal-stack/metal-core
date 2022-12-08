package core

import (
	"github.com/metal-stack/metal-go/api/models"
	"net"
	"time"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-core/cmd/internal/switcher"
	metalgo "github.com/metal-stack/metal-go"
	"go.uber.org/zap"
)

type NOS interface {
	Apply(cfg *switcher.Conf) error
	GetNics(log *zap.SugaredLogger, blacklist []string) ([]*models.V1SwitchNic, error)
	GetSwitchPorts() ([]*net.Interface, error)
}

type Core struct {
	log      *zap.SugaredLogger
	logLevel string

	cidr                      string
	loopbackIP                string
	asn                       string
	partitionID               string
	rackID                    string
	enableReconfigureSwitch   bool
	reconfigureSwitchInterval time.Duration
	managementGateway         string
	additionalBridgePorts     []string
	additionalBridgeVIDs      []string
	spineUplinks              string
	dhcpServers               []string

	nos NOS

	driver             metalgo.Client
	eventServiceClient v1.EventServiceClient
}

type Config struct {
	Log      *zap.SugaredLogger
	LogLevel string

	CIDR                      string
	LoopbackIP                string
	ASN                       string
	PartitionID               string
	RackID                    string
	ReconfigureSwitch         bool
	ReconfigureSwitchInterval time.Duration
	ManagementGateway         string
	AdditionalBridgePorts     []string
	AdditionalBridgeVIDs      []string
	SpineUplinks              string
	DHCPServers               []string

	NOS NOS

	Driver             metalgo.Client
	EventServiceClient v1.EventServiceClient
}

func New(c Config) *Core {
	return &Core{
		log:                       c.Log,
		logLevel:                  c.LogLevel,
		cidr:                      c.CIDR,
		loopbackIP:                c.LoopbackIP,
		asn:                       c.ASN,
		partitionID:               c.PartitionID,
		rackID:                    c.RackID,
		enableReconfigureSwitch:   c.ReconfigureSwitch,
		reconfigureSwitchInterval: c.ReconfigureSwitchInterval,
		managementGateway:         c.ManagementGateway,
		additionalBridgePorts:     c.AdditionalBridgePorts,
		additionalBridgeVIDs:      c.AdditionalBridgeVIDs,
		spineUplinks:              c.SpineUplinks,
		dhcpServers:               c.DHCPServers,
		nos:                       c.NOS,
		driver:                    c.Driver,
		eventServiceClient:        c.EventServiceClient,
	}
}
