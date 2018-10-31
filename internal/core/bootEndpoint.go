package core

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	log "github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type BootResponse struct {
	Kernel      string   `json:"kernel"`
	InitRamDisk []string `json:"initrd"`
	CommandLine string   `json:"cmdline"`
}

func bootEndpoint(request *restful.Request, response *restful.Response) {
	mac := request.PathParameter("mac")

	log.WithField("mac", mac).
		Info("Request metal API for a device with given mac")

	sc, devs := srv.GetMetalAPIClient().FindDevices(mac)

	if sc == http.StatusOK {
		if len(devs) == 0 {
			log.WithField("statusCode", sc).
				Info("Device(s) not found")
			rest.Respond(response, http.StatusOK, createBootDiscoveryImageResponse())
		} else {
			log.WithFields(log.Fields{
				"apiStatusCode": sc,
				"mac":           mac,
			}).Error("There should not exist a device with given mac")
			rest.Respond(response, http.StatusAccepted, createBootTinyCoreLinuxResponse())
		}
	} else {
		log.WithFields(log.Fields{
			"apiStatusCode": sc,
			"mac":           mac,
		}).Error("Failed to request Metal-API")
		rest.Respond(response, http.StatusBadRequest, createBootTinyCoreLinuxResponse())
	}
}

func createBootDiscoveryImageResponse() BootResponse {
	cmdLine := "console=tty0"

	blobstore := "https://blobstore.fi-ts.io/metal/images"
	prefix := srv.GetConfig().HammerImagePrefix
	kernel := fmt.Sprintf("%s/%s-kernel", blobstore, prefix)
	ramdisk := fmt.Sprintf("%s/%s-initrd.img.lz4", blobstore, prefix)
	cmdlineSource := fmt.Sprintf("%s/%s-cmdline", blobstore, prefix)

	if resp, err := resty.R().Get(cmdlineSource); err != nil {
		log.WithFields(log.Fields{
			"err":           err,
			"cmdlineSource": cmdlineSource,
		}).
			Error("could not retrieve cmdline source")
	} else {
		cmdLine = string(resp.Body())
	}
	if len(cmdLine) > 0 {
		cmdLine += " "
	}
	cmdLine += fmt.Sprintf("METAL_CORE_URL=http://%v:%d", srv.GetConfig().ControlPlaneIP, srv.GetConfig().Port)
	return BootResponse{
		Kernel: kernel,
		InitRamDisk: []string{
			ramdisk,
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
