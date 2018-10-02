package metalcore

import (
	"encoding/json"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
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
	// GIVEN
	go func() {
		runMetalAPIServerMock()
	}()

	go func() {
		runMetalcoreServer(t)
	}()

	time.Sleep(200 * time.Millisecond)

	bootResponse := BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img",
		},
		CommandLine: "console=tty0 console=ttyS0",
	}
	var expected []byte
	if m, err := json.Marshal(bootResponse); err != nil {
		assert.Fail(t, "Marshalling should not fail")
	} else {
		expected = m
	}

	// WHEN
	response, err := fakePXEBootRequest()

	// THEN
	if err != nil {
		assert.Fail(t, "Valid PXE boot response expected", "\nExpected: %v\nActual: %v", rest.BytesToString(expected), err)
	} else {
		assert.Equal(t, rest.BytesToString(expected), strings.TrimSpace(rest.BytesToString(response.Body())))
		assert.Equal(t, http.StatusOK, response.StatusCode())
	}
}

func runMetalcoreServer(t *testing.T) {
	config := domain.Config{
		ServerAddress: "localhost",
		ServerPort:    4242,
	}
	if err := envconfig.Process("metalcore", &config); err != nil {
		assert.Fail(t, "Cannot fetch configuration")
	}
	NewService(config).RunServer()
}

func runMetalAPIServerMock() {
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

func fakePXEBootRequest() (*resty.Response, error) {
	return resty.R().Get("http://localhost:4242/v1/boot/fake-mac")
}
