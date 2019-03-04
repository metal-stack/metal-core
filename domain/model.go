package domain

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/emicklei/go-restful"
)

type EventType string

type MachineEvent struct {
	Type EventType            `json:"type,omitempty"`
	Old  *models.MetalMachine `json:"old,omitempty"`
	New  *models.MetalMachine `json:"new,omitempty"`
	SwitchID string           `json:"switchID,omitempty"`
}

// Some EventType enums.
const (
	Create EventType = "create"
	Update EventType = "update"
	Delete EventType = "delete"
)

type APIClient interface {
	FindMachines(mac string) (int, []*models.MetalMachine)
	RegisterMachine(machineId string, request *MetalHammerRegisterMachineRequest) (int, *models.MetalMachine)
	InstallImage(machineId string) (int, *models.MetalMachineWithPhoneHomeToken)
	IPMIConfig(machineId string) (*IPMIConfig, error)
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
	FreeMachine(machine *models.MetalMachine)
	ReconfigureSwitch(switchID string)
}

type Config struct {
	// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
	IP                string `required:"true" desc:"set the metal core IP"`
	PartitionID       string `required:"true" desc:"set the partition ID" envconfig:"partition_id"`
	RackID            string `required:"true" desc:"set the rack ID" envconfig:"rack_id"`
	BindAddress       string `required:"false" default:"0.0.0.0" desc:"set server bind address" split_words:"true"`
	Port              int    `required:"false" default:"4242" desc:"set server port"`
	LogLevel          string `required:"false" default:"info" desc:"set log level" split_words:"true"`
	ConsoleLogging    bool   `required:"false" default:"true" desc:"enable/disable console logging" split_words:"true"`
	ApiProtocol   string `required:"false" default:"http" desc:"set metal api protocol" envconfig:"metal_api_protocol"`
	ApiIP         string `required:"false" default:"localhost" desc:"set metal api address" envconfig:"metal_api_ip"`
	ApiPort       int    `required:"false" default:"8080" desc:"set metal api port" envconfig:"metal_api_port"`
	MQAddress     string `required:"false" default:"localhost:4161" desc:"set the MQ server address" envconfig:"mq_address"`
	MachineTopic  string `required:"false" default:"machine" desc:"set the machine topic name" split_words:"true"`
	LoopbackIP    string `required:"false" default:"10.0.0.11" desc:"set the loopback ip address that is used with BGP unnumbered" envconfig:"metal_loopback_ip"`
	ASN           string `required:"false" default:"420000011" desc:"set the ASN that is used with BGP"`
	SpineUplinks  string `required:"false" default:"swp31,swp32" desc:"set the ports that are connected to spines" split_words:"true"`
	ReconfigureSwitch bool   `required:"false" default:"false" desc:"let metal-core reconfigure the switch" split_words:"true"`
}

type BootConfig struct {
	MetalHammerImageURL    string
	MetalHammerKernelURL   string
	MetalHammerCommandLine string
}

type IPMIConfig struct {
	Hostname  string
	Interface string
	Port      int
	Username  string
	Password  string
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
