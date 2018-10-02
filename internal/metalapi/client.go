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
		InstallImage(deviceUuid string) (int, domain.Image)
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
	statusCode := c.get("/device/find", rest.CreateParams(rest.QueryParameters, "mac", mac), &devices)
	return statusCode, devices
}

func (c client) RegisterDevice(deviceUuid string, lshw []byte) (int, domain.Device) {
	request := domain.RegisterDeviceRequest{
		UUID:       deviceUuid,
		Macs:       []string{},
		FacilityID: "NBG1",
		SizeID:     "t1.small.x86",
	}
	//TODO populate request with appropriate values from lshw
	var device domain.Device
	statusCode := c.post("/device/register", nil, request, &device)
	return statusCode, device
}

func (c client) InstallImage(deviceUuid string) (int, domain.Image) {
	var image domain.Image
	statusCode := c.get("/image", rest.CreateParams(rest.PathParameters, "id", "Alpine 3.8"), &image)
	return statusCode, image
}

func (c client) ReportDeviceState(deviceUuid string, state string) int {
	body := ""
	//TODO populate body appropriately
	statusCode := c.postWithoutResponse("/device/report", nil, body)
	return statusCode
}

func (c client) GetSwitchPorts(deviceUuid string) (int, []domain.SwitchPort) {
	body := ""
	var switchPorts []domain.SwitchPort
	//TODO populate body appropriately
	statusCode := c.post("/device/switch-ports", nil, body, &switchPorts)
	return statusCode, switchPorts
}

func (c client) Ready(deviceUuid string) int {
	body := ""
	//TODO populate body appropriately
	statusCode := c.postWithoutResponse("/device/ready", nil, body)
	return statusCode
}

func (c client) get(path string, params *rest.Params, domainObject interface{}) int {
	return rest.Get(c.Config.MetalApiProtocol, c.Config.MetalApiAddress, c.Config.MetalApiPort, path, params, domainObject)
}

func (c client) post(path string, params *rest.Params, body interface{}, domainObject interface{}) int {
	return rest.Post(c.Config.MetalApiProtocol, c.Config.MetalApiAddress, c.Config.MetalApiPort, path, params, body, domainObject)
}

func (c client) postWithoutResponse(path string, params *rest.Params, body interface{}) int {
	return rest.PostWithoutResponse(c.Config.MetalApiProtocol, c.Config.MetalApiAddress, c.Config.MetalApiPort, path, params, body)
}
