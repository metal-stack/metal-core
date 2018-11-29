package domain

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	sw "git.f-i-ts.de/cloud-native/metal/metal-core/client/switch_operations"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/emicklei/go-restful"
)

type (
	APIClient interface {
		FindDevices(mac string) (int, []*models.MetalDevice)
		RegisterDevice(deviceId string, request *MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice)
		InstallImage(deviceId string) (int, *models.MetalDeviceWithPhoneHomeToken)
		IPMIData(deviceId string) (*IpmiConnection, error)
	}

	Server interface {
		Run()
	}

	Endpoint interface {
		NewBootService() *restful.WebService
		NewDeviceService() *restful.WebService

		Boot(request *restful.Request, response *restful.Response)
		Install(request *restful.Request, response *restful.Response)
		Register(request *restful.Request, response *restful.Response)
		Report(request *restful.Request, response *restful.Response)
	}

	EventHandler interface {
		FreeDevice(device *models.MetalDevice)
	}

	Config struct {
		// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
		IP                string `required:"true" desc:"set the metal core IP"`
		SiteID            string `required:"true" desc:"set the site ID" split_words:"true"`
		RackID            string `required:"true" desc:"set the rack ID" split_words:"true"`
		BindAddress       string `required:"false" default:"0.0.0.0" desc:"set server bind address" split_words:"true"`
		Port              int    `required:"false" default:"4242" desc:"set server port"`
		LogLevel          string `required:"false" default:"info" desc:"set log level" split_words:"true"`
		ConsoleLogging    bool   `required:"false" default:"true" desc:"enable/disable console logging" split_words:"true"`
		ApiProtocol       string `required:"false" default:"http" desc:"set metal api protocol" envconfig:"metal_api_protocol"`
		ApiIP             string `required:"false" default:"localhost" desc:"set metal api address" envconfig:"metal_api_ip"`
		ApiPort           int    `required:"false" default:"8080" desc:"set metal api port" envconfig:"metal_api_port"`
		HammerImagePrefix string `required:"false" default:"metal-hammer" desc:"set hammer image prefix for kernel, initrd and cmdline download" split_words:"true"`
		MQAddress         string `required:"false" default:"localhost:4161" desc:"set the MQ server address" envconfig:"mq_address"`
	}

	IpmiConnection struct {
		Hostname  string
		Interface string
		Port      int
		Username  string
		Password  string
	}

	AppContext struct {
		*Config
		ApiClientHandler    func(*AppContext) APIClient
		ServerHandler       func(*AppContext) Server
		EndpointHandler     func(*AppContext) Endpoint
		EventHandlerHandler func(*AppContext) EventHandler
		DeviceClient        *device.Client
		SwitchClient        *sw.Client
	}
)

func (a *AppContext) ApiClient() APIClient {
	return a.ApiClientHandler(a)
}

func (a *AppContext) Server() Server {
	return a.ServerHandler(a)
}

func (a *AppContext) Endpoint() Endpoint {
	return a.EndpointHandler(a)
}

func (a *AppContext) EventHandler() EventHandler {
	return a.EventHandlerHandler(a)
}
