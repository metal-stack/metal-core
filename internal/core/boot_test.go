package core

import (
	"encoding/json"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"gopkg.in/resty.v1"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPXEBoot(t *testing.T) {
	// GIVEN
	go func() {
		runMetalAPIServerMock()
	}()

	go func() {
		runMetalCoreServer(t)
	}()

	time.Sleep(200 * time.Millisecond)

	br := BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img",
		},
		CommandLine: "console=tty0 console=ttyS0 METAL_CONTROL_PLANE_IP=localhost",
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
		assert.Fail(t, "Valid PXE boot response expected", "\nExpected: %v\nActual: %v", expected, err)
	} else {
		assert.Equal(t, expected, strings.TrimSpace(string(resp.Body())))
		assert.Equal(t, http.StatusOK, resp.StatusCode())
	}
}

func runMetalCoreServer(t *testing.T) {
	config := domain.Config{
		Address: "localhost",
		Port:    4242,
	}
	if err := envconfig.Process("metal-core", &config); err != nil {
		assert.Fail(t, "Cannot fetch configuration")
	}
	NewService(config).RunServer()
}

func runMetalAPIServerMock() {
	router := mux.NewRouter()
	router.HandleFunc("/device/find", findDeviceMockEndpoint).Methods("GET")

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: http.TimeoutHandler(router, time.Second, "Timeout!"),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func findDeviceMockEndpoint(w http.ResponseWriter, r *http.Request) {
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
