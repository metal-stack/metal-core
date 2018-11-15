package test

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"net/http"
	"testing"
)

var devId = "fake-device-id"

type apiHandlerCoreTest struct{}

func TestLoggingMiddleware(t *testing.T) {
	// GIVEN
	runMetalCoreServer(func(ctx *domain.AppContext) domain.APIClient {
		return apiHandlerCoreTest{}
	})
	defer deleteLogFile()

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

func (a apiHandlerCoreTest) FindDevices(mac string) (int, []*models.MetalDevice) {
	return -1, nil
}

func (a apiHandlerCoreTest) RegisterDevice(deviceId string, request *domain.MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice) {
	dev := models.MetalDevice{
		ID: devId,
	}
	return http.StatusOK, &dev
}

func (a apiHandlerCoreTest) InstallImage(deviceId string) (int, *models.MetalDeviceWithPhoneHomeToken) {
	return -1, nil
}
