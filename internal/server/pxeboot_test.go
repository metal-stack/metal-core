package server

import (
	"encoding/json"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/metal-api"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
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
	// given
	if err := envconfig.Process("metalcore", &metal_api.Config); err != nil {
		assert.Fail(t, "Could not inject config")
	}

	bootResponse := BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img",
		},
		CommandLine: "console=tty0 console=ttyS0",
	}
	var expected []byte
	if m, err := json.Marshal(bootResponse); err != nil {
		assert.Fail(t, "Mashalling should not fail")
	} else {
		expected = m
	}

	go func() {
		runFakeMetalAPIServer()
	}()

	// Run metalcore server
	go func() {
		Run("localhost", 4242)
	}()

	time.Sleep(200 * time.Millisecond)

	// when
	response, err := fakePXEBootRequest()

	// then
	if err != nil {
		assert.Fail(t, "Valid PXE boot response expected", "\nExpected: %v\nActual: %v", string(expected), err)
	} else {
		assert.Equal(t, string(expected), strings.TrimSpace(string(response.Body())))
		assert.Equal(t, http.StatusOK, response.StatusCode())
	}
}

func fakePXEBootRequest() (*resty.Response, error) {
	return resty.R().Get("http://localhost:4242/v1/boot/fake-mac")
}

func runFakeMetalAPIServer() {
	router := mux.NewRouter()
	router.HandleFunc("/device/find", findDeviceMockEndpoint).Methods("GET")

	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: router,
	}

	if err := srv.ListenAndServe(); err != nil {
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
