package int

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"net/http"
	"strings"
	"testing"
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

	brJson, _ := json.Marshal(core.BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img",
		},
		CommandLine: fmt.Sprintf("console=tty0 console=ttyS0 METAL_CORE_URL=http://localhost:%d", srv.GetConfig().Port),
	})
	expected := string(brJson)

	// WHEN
	resp, err := fakePXEBootRequest()

	// THEN
	if err != nil {
		assert.Failf(t, "Valid PXE boot response expected", "\nExpected: %v\nActual: %v", expected, err)
	} else {
		assert.Equal(t, expected, strings.TrimSpace(string(resp.Body())))
		assert.Equal(t, http.StatusOK, resp.StatusCode())
	}
}

func findDevicesAPIEndpointMock(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("mac") == fakeMac {
		rest.Respond(w, http.StatusOK, []domain.Device{})
	} else {
		rest.Respond(w, http.StatusAlreadyReported, []domain.Device{
			{}, // Simulate at least one existing device
		})
	}
}

func fakePXEBootRequest() (*resty.Response, error) {
	return resty.R().Get(fmt.Sprintf("http://localhost:%d/v1/boot/%v", srv.GetConfig().Port, fakeMac))
}
