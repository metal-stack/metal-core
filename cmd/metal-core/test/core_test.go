package test

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/maas/metal-core/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"net/http"
	"testing"
)

var devId = "fake-device-id"

func TestLoggingMiddleware(t *testing.T) {
	// GIVEN
	runMetalCoreServer()
	mockMetalAPIServer(endpoint{
		path:    "/device/register",
		handler: registerDeviceAPIEndpointMock,
		method:  http.MethodPost,
	})
	defer shutdown()

	// WHEN
	registerDevice()

	// THEN
	logs := getLogs()
	assert.Contains(t, logs, "Register device at Metal-API")
	assert.Contains(t, logs, "Device registered")
	assert.NotContains(t, logs, "level=error")
}

func registerDevice() (*resty.Response, error) {
	rdr := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devId,
	}
	rdr.Nics = []*models.MetalNic{}
	rdr.Disks = []*models.MetalBlockDevice{}
	return resty.R().SetBody(rdr).
		Post(fmt.Sprintf("http://localhost:%d/device/register/%v", cfg.Port, devId))
}

func registerDeviceAPIEndpointMock(request *restful.Request, response *restful.Response) {
	dev := models.MetalDevice{
		ID: devId,
	}
	rest.Respond(response, http.StatusOK, dev)
}
