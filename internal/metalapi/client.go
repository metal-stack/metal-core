package metalapi

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
)

type (
	Client interface {
		GetConfig() domain.Config
		FindDevices(mac string) (int, []domain.Device)
		RegisterDevice(deviceUuid string, lshw []byte) (int, domain.Device)
		ReportDeviceState(deviceUuid string, state string) int
		GetSwitchPorts(deviceUuid string) (int, []domain.SwitchPort)
		Ready(deviceUuid string) int
	}
	client struct {
		Config domain.Config
	}
)

func NewClient(config domain.Config) Client {
	return client{
		Config: config,
	}
}

func (c client) GetConfig() domain.Config {
	return c.Config
}

func (c client) FindDevices(mac string) (int, []domain.Device) {
	var devices []domain.Device
	statusCode := c.get("/device/find", rest.CreateQueryParameters("mac", mac), &devices)
	return statusCode, devices
}

func (c client) RegisterDevice(deviceUuid string, lshw []byte) (int, domain.Device) {
	request := domain.RegisterDeviceRequest{
		UUID: deviceUuid,
		Macs: []string{},
	}
	//TODO populate request with appropriate values from lshw
	var device domain.Device
	statusCode := c.post("/device/register", request, &device)
	return statusCode, device
}

func (c client) ReportDeviceState(deviceUuid string, state string) int {
	body := ""
	//TODO populate body appropriately
	statusCode := c.postWithoutResponse("/device/report", body)
	return statusCode
}

func (c client) GetSwitchPorts(deviceUuid string) (int, []domain.SwitchPort) {
	body := ""
	var switchPorts []domain.SwitchPort
	//TODO populate body appropriately
	statusCode := c.post("/device/switch-ports", body, &switchPorts)
	return statusCode, switchPorts
}

func (c client) Ready(deviceUuid string) int {
	body := ""
	//TODO populate body appropriately
	statusCode := c.postWithoutResponse("/device/ready", body)
	return statusCode
}

func (c client) get(path string, query map[string]string, domainObject interface{}) int {
	return rest.Get(c.Config.MetalApiProtocol, c.Config.MetalApiAddress, c.Config.MetalApiPort, path, query, domainObject)
}

func (c client) post(path string, body interface{}, domainObject interface{}) int {
	return rest.Post(c.Config.MetalApiProtocol, c.Config.MetalApiAddress, c.Config.MetalApiPort, path, body, domainObject)
}

func (c client) postWithoutResponse(path string, body interface{}) int {
	return rest.PostWithoutResponse(c.Config.MetalApiProtocol, c.Config.MetalApiAddress, c.Config.MetalApiPort, path, body)
}
