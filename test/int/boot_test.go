package int

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var srv core.Service

func TestPXEBoot(t *testing.T) {
	// GIVEN
	go func() {
		runMetalCoreServer(t, 4242)
	}()
	time.Sleep(100 * time.Millisecond)

	go func() {
		mockFindDevicesAPIEndpoint()
	}()
	time.Sleep(100 * time.Millisecond)

	br := core.BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img",
		},
		CommandLine: "console=tty0 console=ttyS0 METAL_CORE_URL=http://localhost:4242",
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
	return resty.R().Get("http://localhost:4242/v1/boot/fake-mac")
}

func runMetalCoreServer(t *testing.T, evenPort int) {
	os.Setenv("METAL_CORE_CONTROL_PLANE_IP", "localhost")
	os.Setenv("METAL_CORE_FACILITY_ID", "FRA")
	os.Setenv("METAL_CORE_PORT", fmt.Sprintf("%d", evenPort))
	os.Setenv("METAL_CORE_METAL_API_PORT", fmt.Sprintf("%d", evenPort+1))
	config := domain.Config{}
	if err := envconfig.Process("METAL_CORE", &config); err != nil {
		assert.Fail(t, "Cannot fetch configuration")
	}
	srv = core.NewService(&config)
	srv.RunServer()
}

func runMetalAPIMockServer(router *mux.Router) {
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", srv.GetConfig().APIPort),
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
