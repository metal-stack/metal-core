package int

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"net/http"
	"strings"
	"testing"
)

var devId = "fake-device-id"

func TestLoggingMiddleware(t *testing.T) {
	// GIVEN
	runMetalCoreServer()
	mockMetalAPIServer(endpoint{
		path:    "/device/register",
		handler: registerDeviceAPIEndpointMock,
		methods: []string{http.MethodPost},
	})
	defer shutdown()

	// WHEN
	registerDevice()

	// THEN
	result := strings.TrimSpace(logOutput.String())
	assert.Contains(t, result, "Register device at Metal API")
	assert.Contains(t, result, "Device registered")
	assert.NotContains(t, result, "level=error")
}

func registerDevice() (*resty.Response, error) {
	rdr := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devId,
	}
	rdr.Nics = []*models.MetalNic{}
	rdr.Disks = []*models.MetalBlockDevice{}
	return resty.R().SetBody(rdr).
		Post(fmt.Sprintf("http://localhost:%d/device/register/%v", srv.GetConfig().Port, devId))
}

func registerDeviceAPIEndpointMock(w http.ResponseWriter, r *http.Request) {
	dev := models.MetalDevice{
		ID: devId,
	}
	rest.Respond(w, http.StatusOK, dev)
}
