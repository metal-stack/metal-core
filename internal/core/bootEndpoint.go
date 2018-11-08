package core

import (
	"fmt"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	restful "github.com/emicklei/go-restful"
	"go.uber.org/zap"
	resty "gopkg.in/resty.v1"
)

type BootResponse struct {
	Kernel      string   `json:"kernel,omitempty"`
	InitRamDisk []string `json:"initrd"`
	CommandLine string   `json:"cmdline,omitempty"`
}

func bootEndpoint(request *restful.Request, response *restful.Response) {
	mac := request.PathParameter("mac")

	zapup.MustRootLogger().Info("Request Metal-API for a device",
		zap.String("MAC", mac),
	)

	sc, devs := srv.API().FindDevices(mac)

	if sc == http.StatusOK {
		if len(devs) == 0 {
			zapup.MustRootLogger().Info("Device(s) not found",
				zap.Int("statusCode", sc),
				zap.String("MAC", mac),
			)
			rest.Respond(response, http.StatusOK, createBootDiscoveryImageResponse())
		} else {
			zapup.MustRootLogger().Error("There should not exist a device",
				zap.Int("statusCode", sc),
				zap.String("MAC", mac),
			)
			rest.Respond(response, http.StatusAccepted, createBootTinyCoreLinuxResponse())
		}
	} else {
		zapup.MustRootLogger().Error("Failed to request Metal-API for a device",
			zap.Int("apiStatusCode", sc),
			zap.String("MAC", mac),
		)
		rest.Respond(response, http.StatusBadRequest, createBootTinyCoreLinuxResponse())
	}
}

func createBootDiscoveryImageResponse() BootResponse {
	blobstore := "https://blobstore.fi-ts.io/metal/images"
	cfg := srv.Config()
	prefix := cfg.HammerImagePrefix
	kernel := fmt.Sprintf("%s/%s-kernel", blobstore, prefix)
	ramdisk := fmt.Sprintf("%s/%s-initrd.img.lz4", blobstore, prefix)
	cmdlineSource := fmt.Sprintf("%s/%s-cmdline", blobstore, prefix)
	cmdlineOptions := []string{}

	if resp, err := resty.R().Get(cmdlineSource); err != nil {
		zapup.MustRootLogger().Error("Could not retrieve cmdline source",
			zap.String("cmdlineSource", cmdlineSource),
			zap.Error(err),
		)
		cmdlineOptions = append(cmdlineOptions, "console=tty0")
	} else {
		cmdlineOptions = append(cmdlineOptions, string(resp.Body()))
	}

	metalCoreAddress := fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d", cfg.IP, cfg.Port)
	metalAPIURL := fmt.Sprintf("METAL_API_URL=%s://%s:%d", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort)
	cmdlineOptions = append(cmdlineOptions, metalCoreAddress, metalAPIURL)

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		cmdlineOptions = append(cmdlineOptions, "DEBUG=1")
	}
	return BootResponse{
		Kernel: kernel,
		InitRamDisk: []string{
			ramdisk,
		},
		CommandLine: strings.Join(cmdlineOptions, " "),
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
