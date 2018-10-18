package int

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"gopkg.in/resty.v1"
)

var fakeMac = "00:11:22:33:44:55"

func TestPXEBoot(t *testing.T) {
	// GIVEN
	runMetalCoreServer()
	mockMetalAPIServer(endpoint{
		path:    "/device/find",
		handler: findDevicesAPIEndpointMock,
		methods: []string{http.MethodGet},
	})
	defer shutdown()

	expected := core.BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img.gz",
		},
		CommandLine: fmt.Sprintf("METAL_CORE_URL=http://localhost:%d", srv.GetConfig().Port),
	}

	// WHEN
	resp, err := fakePXEBootRequest()

	// THEN
	bootResponse := &core.BootResponse{}
	if err != nil {
		assert.Failf(t, "Invalid boot response: %v", err.Error())
	} else if err := json.Unmarshal(resp.Body(), bootResponse); err!=nil{
		assert.Failf(t, "Invalid boot response: %v", string(resp.Body()))
	} else {
		assert.Equal(t, expected.Kernel, bootResponse.Kernel)
		assert.Equal(t, expected.InitRamDisk, bootResponse.InitRamDisk)
		bootResponse.CommandLine = bootResponse.CommandLine[strings.Index(bootResponse.CommandLine, "METAL_CORE_URL"):]
		assert.Equal(t, expected.CommandLine, bootResponse.CommandLine)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
	}
}

func findDevicesAPIEndpointMock(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("mac") == fakeMac {
		rest.Respond(w, http.StatusOK, []models.MetalDevice{})
	} else {
		rest.Respond(w, http.StatusAlreadyReported, []models.MetalDevice{
			{}, // Simulate at least one existing device
		})
	}
}

func fakePXEBootRequest() (*resty.Response, error) {
	return resty.R().Get(fmt.Sprintf("http://localhost:%d/v1/boot/%v", srv.GetConfig().Port, fakeMac))
}
