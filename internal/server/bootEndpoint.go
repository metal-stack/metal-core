package server

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/metal-api"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func bootEndpoint(w http.ResponseWriter, r *http.Request) {
	mac := mux.Vars(r)["mac"]

	log.Infof("Request metal API for a device with mac \"%v\"", mac)

	statusCode, device := metal_api.FindDevice(mac)

	var response interface{}
	if statusCode == http.StatusOK {
		device.Log()
		log.Error("Device should not be available")
		response = createBootNothingResponse()
	} else {
		log.WithField("statusCode", statusCode).
			Info("Device not found")
		response = createBootDiscoveryImage()
	}

	rest.Respond(w, statusCode, response)
}

func createBootDiscoveryImage() interface{} {
	cmdLine, err := http.Get("https://blobstore.fi-ts.io/metal/images/pxeboot-cmdline")
	return struct {
		Kernel      string   `json:"kernel"`
		InitRamDisk []string `json:"initrd"`
		CommandLine string   `json:"cmdline"`
	}{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img",
		},
		CommandLine: cmdLine,
	}
}

func createBootNothingResponse() interface{} {
	return struct {
		Kernel      string   `json:"kernel"`
		InitRamDisk []string `json:"initrd"`
		CommandLine string   `json:"cmdline"`
	}{
		Kernel: "file:///images/pxeboot-kernel",
		InitRamDisk: []string{
			"file:///images/pxeboot-initrd.img",
		},
		CommandLine: "console=tty0",
	}
}
