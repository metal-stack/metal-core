package endpoint

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func (e endpoint) Boot(request *restful.Request, response *restful.Response) {
	mac := request.PathParameter("mac")

	zapup.MustRootLogger().Info("Request Metal-API for a device",
		zap.String("MAC", mac),
	)

	sc, devs := e.ApiClient().FindDevices(mac)

	if sc == http.StatusOK {
		if len(devs) == 0 {
			zapup.MustRootLogger().Info("Device(s) not found",
				zap.Int("statusCode", sc),
				zap.String("MAC", mac),
			)
			rest.Respond(response, http.StatusOK, createBootDiscoveryImageResponse(e.Config))
			return
		}

		zapup.MustRootLogger().Error("There should not exist a device",
			zap.Int("statusCode", sc),
			zap.String("MAC", mac),
		)
		rest.Respond(response, http.StatusAccepted, createBootTinyCoreLinuxResponse())
		return
	}

	zapup.MustRootLogger().Error("Failed to request Metal-API for a device",
		zap.Int("apiStatusCode", sc),
		zap.String("MAC", mac),
	)
	rest.Respond(response, http.StatusBadRequest, createBootTinyCoreLinuxResponse())
}

func createBootDiscoveryImageResponse(cfg *domain.Config) domain.BootResponse {
	blobstore := "https://blobstore.fi-ts.io/metal/images/metal-hammer"
	prefix := cfg.HammerImagePrefix
	kernel := fmt.Sprintf("%s/%s-kernel", blobstore, prefix)
	ramdisk := fmt.Sprintf("%s/%s-initrd.img.lz4", blobstore, prefix)
	metalCoreAddress := fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d", cfg.IP, cfg.Port)
	metalAPIURL := fmt.Sprintf("METAL_API_URL=%s://%s:%d", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort)
	cmdlineOptions := []string{
		"console=tty0",
		"console=ttyS0",
		"ip=dhcp",
	}
	cmdlineOptions = append(cmdlineOptions, metalCoreAddress, metalAPIURL)
	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		cmdlineOptions = append(cmdlineOptions, "DEBUG=1")
	}

	return domain.BootResponse{
		Kernel: kernel,
		InitRamDisk: []string{
			ramdisk,
		},
		CommandLine: strings.Join(cmdlineOptions, " "),
	}
}

func createBootTinyCoreLinuxResponse() domain.BootResponse {
	return domain.BootResponse{
		Kernel: "http://tinycorelinux.net/7.x/x86/release/distribution_files/vmlinuz64",
		InitRamDisk: []string{
			"http://tinycorelinux.net/7.x/x86/release/distribution_files/rootfs.gz",
			"http://tinycorelinux.net/7.x/x86/release/distribution_files/modules64.gz",
		},
		CommandLine: "console=tty0",
	}
}
