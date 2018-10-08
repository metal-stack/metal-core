package api

import (
	"gopkg.in/resty.v1"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
)

type (
	Client interface {
		GetConfig() *domain.Config
		FindDevices(mac string) (int, []domain.Device)
		RegisterDevice(deviceId string, hw []byte) (int, *domain.Device)
		InstallImage(deviceId string) (int, *domain.Device)
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

func (c client) getExpect(path string, queryParams *rest.QueryParams, v interface{}) int {
	if resp := c.newRequest(path, queryParams).Get(); resp != nil {
		rest.Unmarshal(resp, v)
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) postExpect(path string, queryParams *rest.QueryParams, body interface{}, v interface{}) int {
	if resp := c._post(path, queryParams, body); resp != nil {
		rest.Unmarshal(resp, v)
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) post(path string, queryParams *rest.QueryParams, body interface{}) int {
	if resp := c._post(path, queryParams, body); resp != nil {
		return resp.StatusCode()
	} else {
		return http.StatusInternalServerError
	}
}

func (c client) _post(path string, queryParams *rest.QueryParams, body interface{}) *resty.Response {
	if bodyJson := rest.Marshal(body); len(bodyJson) == 0 {
		return nil
	} else {
		return c.newRequest(path, queryParams).Post(bodyJson)
	}
}

func (c client) newRequest(path string, queryParams *rest.QueryParams) *rest.Request {
	return rest.NewRequest(c.Config.APIProtocol, c.Config.APIAddress, c.Config.APIPort, path, queryParams)
}
