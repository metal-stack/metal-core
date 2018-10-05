package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"net/http"
)

type (
	Client interface {
		GetConfig() *domain.Config
		FindDevices(mac string) (int, []domain.Device)
		RegisterDevice(deviceId string, hw []byte) (int, *domain.Device)
		InstallImage(deviceId string) (int, *domain.Image)
		ReportDeviceState(deviceId string, state string) int
		GetSwitchPorts(deviceId string) (int, []domain.SwitchPort)
		Ready(deviceId string) int
	}
	client struct {
		Config *domain.Config
	}
)

func NewClient(cfg *domain.Config) Client {
	return client{
		Config: cfg,
	}
}

func (c client) GetConfig() *domain.Config {
	return c.Config
}

func (c client) ReportDeviceState(deviceId string, state string) int {
	body := ""
	//TODO populate body appropriately
	sc := c.post("/device/report", nil, body)
	return sc
}

func (c client) GetSwitchPorts(deviceId string) (int, []domain.SwitchPort) {
	body := ""
	var sp []domain.SwitchPort
	//TODO populate body appropriately
	sc := c.postExpect("/device/switch-ports", nil, body, &sp)
	return sc, sp
}

func (c client) Ready(deviceId string) int {
	body := ""
	//TODO populate body appropriately
	sc := c.post("/device/ready", nil, body)
	return sc
}

func (c client) getExpect(path string, params *rest.Params, v interface{}) int {
	if resp := c.newRequest(path, params).Get(); resp != nil {
		rest.Unmarshal(resp, v)
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) postExpect(path string, params *rest.Params, body interface{}, v interface{}) int {
	if resp := c.newRequest(path, params).Post(body); resp != nil {
		rest.Unmarshal(resp, v)
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) post(path string, params *rest.Params, body interface{}) int {
	if resp := c.newRequest(path, params).Post(body); resp != nil {
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) newRequest(path string, params *rest.Params) *rest.Request {
	return rest.NewRequest(c.Config.APIProtocol, c.Config.APIAddress, c.Config.APIPort, path, params)
}
