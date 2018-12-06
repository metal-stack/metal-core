package test

import (
	"fmt"
	"net/http"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
)

var devId = "fake-device-id"

type apiHandlerCoreTest struct{}

func TestLoggingMiddleware(t *testing.T) {
	// GIVEN
	e := mockApiEndpoint(func(ctx *domain.AppContext) domain.APIClient {
		return apiHandlerCoreTest{}
	})
	defer deleteLogFile()

	restful.Add(e.NewDeviceService())

	payload := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devId,
	}
	payload.Nics = []*models.MetalNic{}
	payload.Disks = []*models.MetalBlockDevice{}

	// WHEN
	sc := doPost(fmt.Sprintf("/device/register/%v", devId), payload)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	logs := getLogs()
	require.Contains(t, logs, "Register device at Metal-API")
	require.Contains(t, logs, "Device registered")
	require.NotContains(t, logs, "level=error")
}

func (a apiHandlerCoreTest) FindDevices(mac string) (int, []*models.MetalDevice) {
	return -1, nil
}

func (a apiHandlerCoreTest) RegisterDevice(deviceId string, request *domain.MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice) {
	dev := models.MetalDevice{
		ID: &devId,
	}
	return http.StatusOK, &dev
}

func (a apiHandlerCoreTest) InstallImage(deviceId string) (int, *models.MetalDeviceWithPhoneHomeToken) {
	return -1, nil
}

func (a apiHandlerCoreTest) IPMIData(deviceId string) (*domain.IpmiConnection, error) {
	return nil, nil
}
