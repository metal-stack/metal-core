package api

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"net/http"
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

func NewClient(cfg domain.Config) Client {
	return client{
		Config: cfg,
	}
}

func (c client) GetConfig() domain.Config {
	return c.Config
}

func (c client) FindDevices(mac string) (int, []domain.Device) {
	var devs []domain.Device
	sc := c.get("/device/find", rest.CreateQueryParams("mac", mac), &devs)
	return sc, devs
}

func (c client) RegisterDevice(deviceUuid string, lshw []byte) (int, domain.Device) {
	req := domain.RegisterDeviceRequest{
		UUID:       deviceUuid,
		Macs:       []string{},
		FacilityID: "NBG1",
		SizeID:     "t1.small.x86",
	}
	//TODO populate request with appropriate values from lshw
	var dev domain.Device
	sc := c.post("/device/register", nil, req, &dev)
	return sc, dev
}

func (c client) InstallImage(deviceUuid string) (int, domain.Image) {
	var img domain.Image
	sc := c.get(fmt.Sprintf("/image/%v", "2"), nil, &img)
	return sc, img
}

func (c client) ReportDeviceState(deviceUuid string, state string) int {
	b := ""
	//TODO populate body appropriately
	sc := c.postWithoutResponse("/device/report", nil, b)
	return sc
}

func (c client) GetSwitchPorts(deviceUuid string) (int, []domain.SwitchPort) {
	b := ""
	var sp []domain.SwitchPort
	//TODO populate body appropriately
	sc := c.post("/device/switch-ports", nil, b, &sp)
	return sc, sp
}

func (c client) Ready(deviceUuid string) int {
	b := ""
	//TODO populate body appropriately
	sc := c.postWithoutResponse("/device/ready", nil, b)
	return sc
}

func (c client) get(path string, params *rest.Params, v interface{}) int {
	if resp := c.newRequest(path, params).Get(); resp != nil {
		rest.Unmarshal(resp, v)
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) post(path string, params *rest.Params, body interface{}, v interface{}) int {
	if resp := c.newRequest(path, params).Post(body); resp != nil {
		rest.Unmarshal(resp, v)
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) postWithoutResponse(path string, params *rest.Params, body interface{}) int {
	if resp := c.newRequest(path, params).Post(body); resp != nil {
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) newRequest(path string, params *rest.Params) *rest.Request {
	return rest.NewRequest(c.Config.APIProtocol, c.Config.APIAddress, c.Config.APIPort, path, params)
}
