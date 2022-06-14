package api

import (
	"time"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	metalgo "github.com/metal-stack/metal-go"
	"go.uber.org/zap"
)

type ApiClient struct {
	log      *zap.Logger
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

	interfacesTplFile string
	frrTplFile        string

	driver             metalgo.Client
	eventServiceClient v1.EventServiceClient
}

type Config struct {
	Log      *zap.Logger
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

	InterfacesTplFile string
	FrrTplFile        string

	Driver             metalgo.Client
	EventServiceClient v1.EventServiceClient
}

func New(c Config) *ApiClient {
	return &ApiClient{
		log:                c.Log,
		driver:             c.Driver,
		eventServiceClient: c.EventServiceClient,
	}
}
