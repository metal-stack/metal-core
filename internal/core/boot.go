package core

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
	"net/http"
)

type BootResponse struct {
	Kernel      string   `json:"kernel"`
	InitRamDisk []string `json:"initrd"`
	CommandLine string   `json:"cmdline"`
}

var Config domain.Config

func bootEndpoint(w http.ResponseWriter, r *http.Request) {
	mac := mux.Vars(r)["mac"]

	log.WithField("mac", mac).
		Info("Request metal API for a device with given mac")

	sc, devs := srv.GetMetalAPIClient().FindDevices(mac)

	if sc == http.StatusOK && len(devs) == 0 {
		log.WithField("statusCode", sc).
			Info("Device(s) not found")
		rest.Respond(w, http.StatusOK, createBootDiscoveryImageResponse())
	} else {
		log.WithFields(log.Fields{
			"statusCode": sc,
			"mac":        mac,
		}).Error("There should not exist a device with given mac")
		rest.Respond(w, http.StatusAccepted, createBootTinyCoreLinuxResponse())
	}
}

func createBootDiscoveryImageResponse() BootResponse {
	cmdLine := "console=tty0"
	if resp, err := resty.R().Get("https://blobstore.fi-ts.io/metal/images/pxeboot-cmdline"); err != nil {
		log.WithField("err", err).
			Error("File 'pxeboot-cmdline' could not be retrieved")
	} else {
		cmdLine = string(resp.Body())
	}
	cmdLine += fmt.Sprintf(" METAL_CONTROL_PLANE_IP=%v", Config.ControlPlaneIP)
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
