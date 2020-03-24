package domain

import (
	"time"

	"github.com/emicklei/go-restful"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/metal-stack/metal-core/client/machine"
	"github.com/metal-stack/metal-core/client/partition"
	sw "github.com/metal-stack/metal-core/client/switch_operations"
	"github.com/metal-stack/metal-core/models"
	"github.com/metal-stack/security"
)

type EventType string

type MachineCommand string

const (
	MachineOnCmd             MachineCommand = "ON"
	MachineOffCmd            MachineCommand = "OFF"
	MachineResetCmd          MachineCommand = "RESET"
	MachineBiosCmd           MachineCommand = "BIOS"
	ChassisIdentifyLEDOnCmd  MachineCommand = "LED-ON"
	ChassisIdentifyLEDOffCmd MachineCommand = "LED-OFF"
)

type MachineExecCommand struct {
	TargetMachineID string         `json:"target,omitempty"`
	Command         MachineCommand `json:"cmd,omitempty"`
	Params          []string       `json:"params,omitempty"`
}

type MachineEvent struct {
	Type         EventType           `json:"type,omitempty"`
	OldMachineID string              `json:"old,omitempty"`
	NewMachineID string              `json:"new,omitempty"`
	Cmd          *MachineExecCommand `json:"cmd,omitempty"`
}

// Some EventType enums.
const (
	Create  EventType = "create"
	Update  EventType = "update"
	Delete  EventType = "delete"
	Command EventType = "command"
)

type APIClient interface {
	FindMachines(mac string) (int, []*models.V1MachineResponse)
	FindPartition(id string) (*models.V1PartitionResponse, error)
	RegisterMachine(machineID string, request *MetalHammerRegisterMachineRequest) (int, *models.V1MachineResponse)
	InstallImage(machineID string) (int, *models.V1MachineResponse)
	IPMIConfig(machineID string) (*IPMIConfig, error)
	AddProvisioningEvent(machineID string, event *models.V1MachineProvisioningEvent) error
	FinalizeAllocation(machineID, consolepassword string) (*machine.FinalizeAllocationOK, error)
	RegisterSwitch() (*models.V1SwitchResponse, error)
	ConstantlyPhoneHome()
	SetChassisIdentifyLEDStateOn(machineID, description string) error
	SetChassisIdentifyLEDStateOff(machineID, description string) error
}

type Server interface {
	Run()
}

type EndpointHandler interface {
	NewBootService() *restful.WebService
	NewMachineService() *restful.WebService

	Boot(request *restful.Request, response *restful.Response)
	Install(request *restful.Request, response *restful.Response)
	Register(request *restful.Request, response *restful.Response)
	Report(request *restful.Request, response *restful.Response)
}

type EventHandler interface {
	FreeMachine(machineID string)
	PowerOnMachine(machineID string)
	PowerOffMachine(machineID string)
	PowerResetMachine(machineID string)
	BootBiosMachine(machineID string)

	PowerOnChassisIdentifyLED(machineID, description string)
	PowerOffChassisIdentifyLED(machineID, description string)

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
	HMACKey                   string        `required:"true" desc:"the preshared key for the hmac calculation" envconfig:"hmac_key"`
}

type BootConfig struct {
	MetalHammerImageURL    string
	MetalHammerKernelURL   string
	MetalHammerCommandLine string
}

type IPMIConfig struct {
	Hostname string
	Port     int
	Ipmi     *models.V1MachineIPMI
}

func (i *IPMIConfig) Address() string {
	return IPMIAddress(i.Ipmi)
}

func (i *IPMIConfig) Interface() string {
	return IPMIInterface(i.Ipmi)
}

func (i *IPMIConfig) Mac() string {
	return IPMIMAC(i.Ipmi)
}

func (i *IPMIConfig) User() string {
	return IPMIUser(i.Ipmi)
}

func (i *IPMIConfig) Password() string {
	return IPMIPassword(i.Ipmi)
}

type AppContext struct {
	*Config
	*BootConfig
	apiClient       func(*AppContext) APIClient
	server          func(*AppContext) Server
	endpointHandler func(*AppContext) EndpointHandler
	eventHandler    func(*AppContext) EventHandler
	MachineClient   *machine.Client
	SwitchClient    *sw.Client
	PartitionClient *partition.Client
	hmac            security.HMACAuth
	Auth            runtime.ClientAuthInfoWriter
	DevMode         bool
}

func (a *AppContext) APIClient() APIClient {
	return a.apiClient(a)
}

func (a *AppContext) SetAPIClient(apiClient func(*AppContext) APIClient) {
	a.apiClient = apiClient
}

func (a *AppContext) Server() Server {
	return a.server(a)
}

func (a *AppContext) SetServer(server func(*AppContext) Server) {
	a.server = server
}

func (a *AppContext) EndpointHandler() EndpointHandler {
	return a.endpointHandler(a)
}

func (a *AppContext) SetEndpointHandler(endpointHandler func(*AppContext) EndpointHandler) {
	a.endpointHandler = endpointHandler
}

func (a *AppContext) EventHandler() EventHandler {
	return a.eventHandler(a)
}

func (a *AppContext) SetEventHandler(eventHandler func(*AppContext) EventHandler) {
	a.eventHandler = eventHandler
}

func (a *AppContext) InitHMAC() {
	a.hmac = security.NewHMACAuth("Metal-Edit", []byte(a.HMACKey))
	a.Auth = runtime.ClientAuthInfoWriterFunc(a.auther)
}

func (a *AppContext) auther(rq runtime.ClientRequest, rg strfmt.Registry) error {
	a.hmac.AddAuthToClientRequest(rq, time.Now())
	return nil
}
