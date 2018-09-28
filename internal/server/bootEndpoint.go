package server

import (
	"io/ioutil"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/metal-api"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func bootEndpoint(w http.ResponseWriter, r *http.Request) {
	mac := mux.Vars(r)["mac"]

	log.WithField("mac", mac).
		Info("Request metal API for a device with given mac")

	statusCode, devices := metal_api.FindDevices(mac)

	var response interface{}
	if len(devices) > 0 {
		log.WithFields(log.Fields{
			"statusCode": statusCode,
			"mac": mac,
		}).Error("There should not exist a device with given mac")
		response = createBootNothingResponse()
	} else {
		log.WithField("statusCode", statusCode).
			Info("Device not found")
		response = createBootDiscoveryImageResponse()
	}

	rest.Respond(w, statusCode, response)
}

func createBootDiscoveryImageResponse() interface{} {
	cmdLine := "console=tty0"
	resp, err := http.Get("https://blobstore.fi-ts.io/metal/images/pxeboot-cmdline")
	if err != nil {
		log.Errorf("pxeboot-cmdline could not be retrieved: %v", err)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("pxeboot-cmdline could not be retrieved: %v", err)
		} else {
			cmdLine = strings.TrimSpace(string(body))
		}
	}
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
