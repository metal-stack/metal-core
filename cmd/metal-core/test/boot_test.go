package test

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"net/http"
	"strings"
	"testing"
)

var fakeMac = "00:11:22:33:44:55"

type apiHandlerBootTest struct{}

func TestPXEBoot(t *testing.T) {
	// GIVEN
	runMetalCoreServer(func(ctx *domain.AppContext) domain.APIClient {
		return apiHandlerBootTest{}
	})
	defer truncateLogFile()

	expected := domain.BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img.lz4",
		},
		CommandLine: fmt.Sprintf("METAL_CORE_ADDRESS=127.0.0.1:%d METAL_API_URL=http://%v:%d", cfg.Port, cfg.ApiIP, cfg.ApiPort),
	}

	// WHEN
	resp, err := fakePXEBootRequest()

	// THEN
	if err != nil {
		assert.Failf(t, "Invalid boot response: %v", err.Error())
	}
	bootResponse := &domain.BootResponse{}
	err = json.Unmarshal(resp.Body(), bootResponse)
	if err != nil {
		assert.Failf(t, "Invalid boot response: %v", string(resp.Body()))
		return
	}
	assert.Equal(t, expected.Kernel, bootResponse.Kernel)
	assert.Equal(t, expected.InitRamDisk, bootResponse.InitRamDisk)
	bootResponse.CommandLine = bootResponse.CommandLine[strings.Index(bootResponse.CommandLine, "METAL_CORE_ADDRESS"):]
	assert.Equal(t, expected.CommandLine, bootResponse.CommandLine)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func fakePXEBootRequest() (*resty.Response, error) {
	return resty.R().Get(fmt.Sprintf("http://127.0.0.1:%d/v1/boot/%v", cfg.Port, fakeMac))
}

func (a apiHandlerBootTest) FindDevices(mac string) (int, []*models.MetalDevice) {
	if mac == fakeMac {
		return http.StatusOK, []*models.MetalDevice{}
	}
	return http.StatusAlreadyReported, []*models.MetalDevice{
		{}, // Simulate at least one existing device
	}
}

func (a apiHandlerBootTest) RegisterDevice(deviceId string, request *domain.MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice) {
	return -1, nil
}

func (a apiHandlerBootTest) InstallImage(deviceId string) (int, *models.MetalDeviceWithPhoneHomeToken) {
	return -1, nil
}
