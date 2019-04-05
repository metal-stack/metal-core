package endpoint

import (
	"fmt"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func (e *endpointHandler) Boot(request *restful.Request, response *restful.Response) {
	mac := request.PathParameter("mac")

	zapup.MustRootLogger().Info("Request Metal-API for a machine",
		zap.String("MAC", mac),
	)

	sc, machines := e.APIClient().FindMachines(mac)

	if sc == http.StatusOK {
		if len(machines) == 0 {
			zapup.MustRootLogger().Info("Machine(s) not found",
				zap.Int("statusCode", sc),
				zap.String("MAC", mac),
			)
			rest.Respond(response, http.StatusOK, createBootDiscoveryImageResponse(e))
		}
		// FIXME this should not happen, we should consider returning a recovery image for digging into to root cause.
	}
}

func createBootDiscoveryImageResponse(e *endpointHandler) domain.BootResponse {
	cfg := e.Config

	metalCoreAddress := fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d", cfg.IP, cfg.Port)
	metalAPIURL := fmt.Sprintf("METAL_API_URL=%s://%s:%d", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort)

	cmdline := []string{e.BootConfig.MetalHammerCommandLine, metalCoreAddress, metalAPIURL}
	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		cmdline = append(cmdline, "DEBUG=1")
	}

	return domain.BootResponse{
		Kernel: e.BootConfig.MetalHammerKernelURL,
		InitRamDisk: []string{
			e.BootConfig.MetalHammerImageURL,
		},
		CommandLine: strings.Join(cmdline, " "),
	}
}
