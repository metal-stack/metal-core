package int

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
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
	mockRegisterDeviceEndpoint()
	defer shutdown()

	// WHEN
	requestPostRegisterDevice()

	// THEN
	assert.Contains(t, strings.TrimSpace(logOutput.String()), "Register device at Metal API")
	assert.Contains(t, strings.TrimSpace(logOutput.String()), "Device registered")
	assert.NotContains(t, strings.TrimSpace(logOutput.String()), "level=error")
}

func requestPostRegisterDevice() (*resty.Response, error) {
	rdr := &domain.RegisterDeviceRequest{
		UUID:  devId,
		Nics:  []domain.Nic{},
		Disks: []domain.BlockDevice{},
	}
	return resty.R().SetBody(rdr).
		Post(fmt.Sprintf("http://localhost:%d/device/register/%v", srv.GetConfig().Port, devId))
}

func mockRegisterDeviceEndpoint() {
	router := mux.NewRouter()
	router.HandleFunc("/device/register", registerDeviceMockEndpoint).Methods(http.MethodPost)
	runMetalAPIMockServer(router)
}

func registerDeviceMockEndpoint(w http.ResponseWriter, r *http.Request) {
	dev := domain.Device{
		ID: devId,
	}
	rest.Respond(w, http.StatusOK, dev)
}
