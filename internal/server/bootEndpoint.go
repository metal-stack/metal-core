package server

import (
	"gopkg.in/resty.v1"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/metal-api"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type BootResponse struct {
	Kernel      string   `json:"kernel"`
	InitRamDisk []string `json:"initrd"`
	CommandLine string   `json:"cmdline"`
}

func bootEndpoint(w http.ResponseWriter, r *http.Request) {
	mac := mux.Vars(r)["mac"]

	log.WithField("mac", mac).
		Info("Request metal API for a device with given mac")

	statusCode, devices := metal_api.FindDevices(mac)

	if statusCode == http.StatusOK && len(devices) == 0 {
		log.WithField("statusCode", statusCode).
			Info("Device not found")
		rest.Respond(w, http.StatusOK, createBootDiscoveryImageResponse())
	} else {
		log.WithFields(log.Fields{
			"statusCode": statusCode,
			"mac":        mac,
		}).Error("There should not exist a device with given mac")
		rest.Respond(w, http.StatusAccepted, createBootTinyCoreLinuxResponse())
	}
}

func createBootDiscoveryImageResponse() BootResponse {
	cmdLine := "console=tty0"
	if response, err := resty.R().Get("https://blobstore.fi-ts.io/metal/images/pxeboot-cmdline"); err != nil {
		log.WithField("err", err).
			Error("File 'pxeboot-cmdline' could not be retrieved")
	} else {
		cmdLine = strings.TrimSpace(string(response.Body()))
	}
	return BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/pxeboot-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/pxeboot-initrd.img",
		},
		CommandLine: cmdLine,
	}
}

func createBootTinyCoreLinuxResponse() BootResponse {
	return BootResponse{
		Kernel: "http://tinycorelinux.net/7.x/x86/release/distribution_files/vmlinuz64",
		InitRamDisk: []string{
			"http://tinycorelinux.net/7.x/x86/release/distribution_files/rootfs.gz",
			"http://tinycorelinux.net/7.x/x86/release/distribution_files/modules64.gz",
		},
		CommandLine: "console=tty0",
	}
}
