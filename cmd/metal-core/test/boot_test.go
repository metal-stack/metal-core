package test

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

var fakeMac = "00:11:22:33:44:55"

type apiHandlerBootTest struct{}

func TestPXEBoot(t *testing.T) {
	// GIVEN
	e := mockApiEndpoint(func(ctx *domain.AppContext) domain.APIClient {
		return apiHandlerBootTest{}
	})
	defer truncateLogFile()

	restful.Add(e.NewBootService())

	expected := domain.BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/metal-hammer/metal-hammer-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/metal-hammer/metal-hammer-initrd.img.lz4",
		},
		CommandLine: fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d METAL_API_URL=http://%v:%d", cfg.IP, cfg.Port, cfg.ApiIP, cfg.ApiPort),
	}

	// WHEN
	bootResponse := &domain.BootResponse{}
	sc := doGet(fmt.Sprintf("/v1/boot/%v", fakeMac), bootResponse)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, expected.Kernel, bootResponse.Kernel)
	require.Equal(t, expected.InitRamDisk, bootResponse.InitRamDisk)
	bootResponse.CommandLine = bootResponse.CommandLine[strings.Index(bootResponse.CommandLine, "METAL_CORE_ADDRESS"):]
	require.Equal(t, expected.CommandLine, bootResponse.CommandLine)
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

func (a apiHandlerBootTest) IPMIData(deviceId string) (*domain.IpmiConnection, error) {
	return nil, nil
}
