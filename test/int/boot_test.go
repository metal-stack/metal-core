package int

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"net/http"
	"strings"
	"testing"
)

func TestPXEBoot(t *testing.T) {
	// GIVEN
	runMetalCoreServer()
	mockFindDevicesAPIEndpoint()
	defer shutdown()

	br := core.BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img",
		},
		CommandLine: fmt.Sprintf("console=tty0 console=ttyS0 METAL_CORE_URL=http://localhost:%d", srv.GetConfig().Port),
	}
	var expected string
	if m, err := json.Marshal(br); err != nil {
		assert.Fail(t, "Marshalling should not fail")
	} else {
		expected = string(m)
	}

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

func mockFindDevicesAPIEndpoint() {
	router := mux.NewRouter()
	router.HandleFunc("/device/find", findDevicesMockEndpoint).Methods(http.MethodGet)
	runMetalAPIMockServer(router)
}

func findDevicesMockEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("mac") == "fake-mac" {
		rest.Respond(w, http.StatusOK, []domain.Device{})
	} else {
		rest.Respond(w, http.StatusAlreadyReported, []domain.Device{
			{
				ID: "fakeDeviceID",
			},
		})
	}
}

func fakePXEBootRequest() (*resty.Response, error) {
	return resty.R().Get(fmt.Sprintf("http://localhost:%d/v1/boot/fake-mac", srv.GetConfig().Port))
}
