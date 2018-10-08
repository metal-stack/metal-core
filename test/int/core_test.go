package int

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
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
	assert.Contains(t, strings.TrimSpace(logOutput.String()), "Register device at Metal API")
	assert.Contains(t, strings.TrimSpace(logOutput.String()), "Device registered")
	assert.NotContains(t, strings.TrimSpace(logOutput.String()), "level=error")
}

func registerDevice() (*resty.Response, error) {
	rdr := &domain.RegisterDeviceRequest{
		UUID: devId,
	}
	rdr.Nics = []domain.Nic{}
	rdr.Disks = []domain.BlockDevice{}
	return resty.R().SetBody(rdr).
		Post(fmt.Sprintf("http://localhost:%d/device/register/%v", srv.GetConfig().Port, devId))
}

func registerDeviceAPIEndpointMock(w http.ResponseWriter, r *http.Request) {
	dev := domain.Device{
		ID: devId,
	}
	rest.Respond(w, http.StatusOK, dev)
}
