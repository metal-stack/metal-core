package core

import (
	"bytes"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"net/http"
	"strings"
	"testing"
	"time"
)

var devId = "fake-device-id"

func TestLoggingMiddleware(t *testing.T) {
	// GIVEN
	var out bytes.Buffer
	log.SetOutput(&out)

	go func() {
		runMetalCoreServer(t, 4244)
	}()
	time.Sleep(100 * time.Millisecond)

	go func() {
		mockRegisterDeviceEndpoint()
	}()
	time.Sleep(100 * time.Millisecond)

	// WHEN
	requestPostRegisterDevice()

	// THEN
	assert.NotContains(t, strings.TrimSpace(out.String()), "level=error")
}

func requestPostRegisterDevice() (*resty.Response, error) {
	rdr := &domain.RegisterDeviceRequest{
		UUID:  devId,
		Nics:  []domain.Nic{},
		Disks: []domain.BlockDevice{},
	}
	return resty.R().SetBody(rdr).
		Post(fmt.Sprintf("http://localhost:4244/device/register/%v", devId))
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
