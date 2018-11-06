package core

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/log"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"gopkg.in/resty.v1"
)

type BootResponse struct {
	Kernel      string   `json:"kernel"`
	InitRamDisk []string `json:"initrd"`
	CommandLine string   `json:"cmdline"`
}

func bootEndpoint(request *restful.Request, response *restful.Response) {
	mac := request.PathParameter("mac")

	log.Get().Info("Request Metal-API for a device",
		zap.String("MAC", mac),
	)

	sc, devs := srv.API().FindDevices(mac)

	if sc == http.StatusOK {
		if len(devs) == 0 {
			log.Get().Info("Device(s) not found",
				zap.Int("statusCode", sc),
				zap.String("MAC", mac),
			)
			rest.Respond(response, http.StatusOK, createBootDiscoveryImageResponse())
		} else {
			log.Get().Error("There should not exist a device",
				zap.Int("statusCode", sc),
				zap.String("MAC", mac),
			)
			rest.Respond(response, http.StatusAccepted, createBootTinyCoreLinuxResponse())
		}
	} else {
		log.Get().Error("Failed to request Metal-API for a device",
			zap.Int("apiStatusCode", sc),
			zap.String("MAC", mac),
		)
		rest.Respond(response, http.StatusBadRequest, createBootTinyCoreLinuxResponse())
	}
}

func createBootDiscoveryImageResponse() BootResponse {
	cmdLine := "console=tty0"

	blobstore := "https://blobstore.fi-ts.io/metal/images"
	prefix := srv.Config().HammerImagePrefix
	kernel := fmt.Sprintf("%s/%s-kernel", blobstore, prefix)
	ramdisk := fmt.Sprintf("%s/%s-initrd.img.lz4", blobstore, prefix)
	cmdlineSource := fmt.Sprintf("%s/%s-cmdline", blobstore, prefix)

	if resp, err := resty.R().Get(cmdlineSource); err != nil {
		log.Get().Error("Could not retrieve cmdline source",
			zap.String("cmdlineSource", cmdlineSource),
			zap.Error(err),
		)
	} else {
		cmdLine = string(resp.Body())
	}
	if len(cmdLine) > 0 {
		cmdLine += " "
	}
	cmdLine += fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d", srv.Config().IP, srv.Config().Port)
	if strings.ToUpper(srv.Config().LogLevel) == "DEBUG" {
		cmdLine += " DEBUG=1"
	}
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
