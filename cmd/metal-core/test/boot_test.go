package test

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"

	"gopkg.in/resty.v1"
)

var fakeMac = "00:11:22:33:44:55"

func TestPXEBoot(t *testing.T) {
	// GIVEN
	runMetalCoreServer()
	mockMetalAPIServer(endpoint{
		path:    "/device/find",
		handler: findDevicesAPIEndpointMock,
		method:  http.MethodGet,
	})
	defer shutdown()

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
	bootResponse := &domain.BootResponse{}
	if err != nil {
		assert.Failf(t, "Invalid boot response: %v", err.Error())
	} else if err := json.Unmarshal(resp.Body(), bootResponse); err != nil {
		assert.Failf(t, "Invalid boot response: %v", string(resp.Body()))
	} else {
		assert.Equal(t, expected.Kernel, bootResponse.Kernel)
		assert.Equal(t, expected.InitRamDisk, bootResponse.InitRamDisk)
		bootResponse.CommandLine = bootResponse.CommandLine[strings.Index(bootResponse.CommandLine, "METAL_CORE_ADDRESS"):]
		assert.Equal(t, expected.CommandLine, bootResponse.CommandLine)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
	}
}

func findDevicesAPIEndpointMock(request *restful.Request, response *restful.Response) {
	if request.QueryParameter("mac") == fakeMac {
		rest.Respond(response, http.StatusOK, []models.MetalDevice{})
	} else {
		rest.Respond(response, http.StatusAlreadyReported, []models.MetalDevice{
			{}, // Simulate at least one existing device
		})
	}
}

func fakePXEBootRequest() (*resty.Response, error) {
	return resty.R().Get(fmt.Sprintf("http://127.0.0.1:%d/v1/boot/%v", cfg.Port, fakeMac))
}
