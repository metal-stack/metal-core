package domain

import (
	"time"

	"github.com/metal-stack/go-hal/pkg/api"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	metalgo "github.com/metal-stack/metal-go"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-go/api/models"
)

type EventType string

type MachineCommand string

const (
	MachineOnCmd             MachineCommand = "ON"
	MachineOffCmd            MachineCommand = "OFF"
	MachineResetCmd          MachineCommand = "RESET"
	MachineCycleCmd          MachineCommand = "CYCLE"
	MachineBiosCmd           MachineCommand = "BIOS"
	MachineDiskCmd           MachineCommand = "DISK"
	MachinePxeCmd            MachineCommand = "PXE"
	MachineReinstallCmd      MachineCommand = "REINSTALL"
	ChassisIdentifyLEDOnCmd  MachineCommand = "LED-ON"
	ChassisIdentifyLEDOffCmd MachineCommand = "LED-OFF"
	UpdateFirmwareCmd        MachineCommand = "UPDATE-FIRMWARE"
)

type MachineExecCommand struct {
	TargetMachineID string         `json:"target,omitempty"`
	Command         MachineCommand `json:"cmd,omitempty"`
	Params          []string       `json:"params,omitempty"`
}

type MachineEvent struct {
	Type         EventType           `json:"type,omitempty"`
	OldMachineID string              `json:"old,omitempty"`
	Cmd          *MachineExecCommand `json:"cmd,omitempty"`
	IPMI         *IPMI               `json:"ipmi,omitempty"`
}

type IPMI struct {
	// Address is host:port of the connection to the ipmi BMC, host can be either a ip address or a hostname
	Address  string `json:"address"`
	User     string `json:"user"`
	Password string `json:"password"`
	Fru      Fru    `json:"fru"`
}

type Fru struct {
	BoardPartNumber string `json:"board_part_number"`
}

// Some EventType enums.
const (
	Create  EventType = "create"
	Update  EventType = "update"
	Delete  EventType = "delete"
	Command EventType = "command"
)

type APIClient interface {
	RegisterSwitch() (*models.V1SwitchResponse, error)
	ConstantlyPhoneHome()
	Send(event *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error)
}

type Server interface {
	Run()
}

type EventHandler interface {
	FreeMachine(event MachineEvent)
	PowerOnMachine(event MachineEvent)
	PowerOffMachine(event MachineEvent)
	PowerResetMachine(event MachineEvent)
	PowerCycleMachine(event MachineEvent)
	PowerBootBiosMachine(event MachineEvent)
	PowerBootDiskMachine(event MachineEvent)
	PowerBootPxeMachine(event MachineEvent)
	ReinstallMachine(event MachineEvent)

	PowerOnChassisIdentifyLED(event MachineEvent)
	PowerOffChassisIdentifyLED(event MachineEvent)

	UpdateBios(revision, description string, s3Cfg *api.S3Config, event MachineEvent)
	UpdateBmc(revision, description string, s3Cfg *api.S3Config, event MachineEvent)

	ReconfigureSwitch()
}

type Config struct {
	// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
	CIDR                      string        `required:"true" desc:"set the metal core CIDR"`
	PartitionID               string        `required:"true" desc:"set the partition ID" envconfig:"partition_id"`
	RackID                    string        `required:"true" desc:"set the rack ID" envconfig:"rack_id"`
	BindAddress               string        `required:"false" default:"0.0.0.0" desc:"set server bind address" split_words:"true"`
	MetricsServerPort         int           `required:"false" default:"2112" desc:"the port of the metrics server" split_words:"true"`
	MetricsServerBindAddress  string        `required:"false" default:"0.0.0.0" desc:"the bind addr of the metrics server" split_words:"true"`
	Port                      int           `required:"false" default:"4242" desc:"set server port"`
	LogLevel                  string        `required:"false" default:"info" desc:"set log level" split_words:"true"`
	ConsoleLogging            bool          `required:"false" default:"true" desc:"enable/disable console logging" split_words:"true"`
	ApiProtocol               string        `required:"false" default:"http" desc:"set metal api protocol" envconfig:"metal_api_protocol"`
	ApiIP                     string        `required:"false" default:"localhost" desc:"set metal api address" envconfig:"metal_api_ip"`
	ApiPort                   int           `required:"false" default:"8080" desc:"set metal api port" envconfig:"metal_api_port"`
	ApiBasePath               string        `required:"false" default:"" desc:"set metal api basepath" envconfig:"metal_api_basepath"`
	MQAddress                 string        `required:"false" default:"localhost:4161" desc:"set the MQ server address" envconfig:"mq_address"`
	MQCACertFile              string        `required:"false" default:"" desc:"the CA certificate file for verifying MQ certificate" envconfig:"mq_ca_cert_file"`
	MQClientCertFile          string        `required:"false" default:"" desc:"the client certificate file for accessing MQ" envconfig:"mq_client_cert_file"`
	MQLogLevel                string        `required:"false" default:"info" desc:"sets the MQ loglevel (debug, info, warn, error)" envconfig:"mq_loglevel"`
	MachineTopic              string        `required:"false" default:"machine" desc:"set the machine topic name" split_words:"true"`
	MachineTopicTTL           int           `required:"false" default:"30000" desc:"sets the TTL in milliseconds for MachineTopic" envconfig:"machine_topic_ttl"`
	LoopbackIP                string        `required:"false" default:"10.0.0.11" desc:"set the loopback ip address that is used with BGP unnumbered" split_words:"true"`
	ASN                       string        `required:"false" default:"420000011" desc:"set the ASN that is used with BGP"`
	SpineUplinks              string        `required:"false" default:"swp31,swp32" desc:"set the ports that are connected to spines" split_words:"true"`
	ManagementGateway         string        `required:"false" default:"192.168.0.1" desc:"the default gateway for the management network" split_words:"true"`
	ReconfigureSwitch         bool          `required:"false" default:"false" desc:"let metal-core reconfigure the switch" split_words:"true"`
	ReconfigureSwitchInterval time.Duration `required:"false" default:"10s" desc:"pull interval to fetch and apply switch configuration" split_words:"true"`
	AdditionalBridgeVIDs      []string      `required:"false" desc:"additional vlan ids that should be configured at the vlan-aware bridge" envconfig:"additional_bridge_vids"`
	AdditionalBridgePorts     []string      `required:"false" desc:"additional switch ports that should be configured at the vlan-aware bridge" envconfig:"additional_bridge_ports"`
	ChangeBootOrder           bool          `required:"false" default:"true" desc:"issue ipmi commands to change boot order" split_words:"true"`
	InterfacesTplFile         string        `required:"false" default:"" desc:"the golang template file used to render /etc/network/interfaces, a default template is included" envconfig:"interfaces_tpl_file"`
	FrrTplFile                string        `required:"false" default:"" desc:"the golang template file used to render /etc/frr/frr.conf, a default template is included" envconfig:"frr_tpl_file"`
	HMACKey                   string        `required:"true" desc:"the preshared key for the hmac calculation" envconfig:"hmac_key"`
	GrpcAddress               string        `required:"true" default:"" desc:"the gRPC address" envconfig:"grpc_address"`
	GrpcCACertFile            string        `required:"false" desc:"the gRPC CA certificate file" envconfig:"grpc_ca_cert_file"`
	GrpcClientCertFile        string        `required:"false" desc:"the gRPC client certificate file" envconfig:"grpc_client_cert_file"`
	GrpcClientKeyFile         string        `required:"false" desc:"the gRPC client key file" envconfig:"grpc_client_key_file"`
}

type BootConfig struct {
	MetalHammerImageURL    string
	MetalHammerKernelURL   string
	MetalHammerCommandLine string
}

type AppContext struct {
	*Config
	*BootConfig
	apiClient          func(*AppContext) APIClient
	server             func(*AppContext) Server
	eventHandler       func(*AppContext) EventHandler
	Driver             metalgo.Client
	EventServiceClient v1.EventServiceClient
	Log                *zap.Logger
}

func (a *AppContext) APIClient() APIClient {
	return a.apiClient(a)
}

func (a *AppContext) SetAPIClient(apiClient func(*AppContext) APIClient) {
	a.apiClient = apiClient
}

func (a *AppContext) SetEventServiceClient(eventServiceClient v1.EventServiceClient) {
	a.EventServiceClient = eventServiceClient
}

func (a *AppContext) Server() Server {
	return a.server(a)
}

func (a *AppContext) SetServer(server func(*AppContext) Server) {
	a.server = server
}

func (a *AppContext) EventHandler() EventHandler {
	return a.eventHandler(a)
}

func (a *AppContext) SetEventHandler(eventHandler func(*AppContext) EventHandler) {
	a.eventHandler = eventHandler
}
