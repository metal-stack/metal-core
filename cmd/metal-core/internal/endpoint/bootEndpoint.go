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
			rest.Respond(response, http.StatusOK, createBootDiscoveryImageResponse(&e))
		}
		// FIXME this should not happen, we should consider returning a recovery image for digging into to root cause.
	}
}

func createBootDiscoveryImageResponse(e *endpoint) domain.BootResponse {
	cfg := e.Config

	metalCoreAddress := fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d", cfg.IP, cfg.Port)
	metalAPIURL := fmt.Sprintf("METAL_API_URL=%s://%s:%d", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort)

	cmdline := e.BootConfig.MetalHammerCommandLine
	cmdline += " " + metalCoreAddress
	cmdline += " " + metalAPIURL

	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		cmdline += " " + "DEBUG=1"
	}

	return domain.BootResponse{
		Kernel: e.BootConfig.MetalHammerKernelURL,
		InitRamDisk: []string{
			e.BootConfig.MetalHammerImageURL,
		},
		CommandLine: cmdline,
	}
}
