package endpoint

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"
)

type bootConfig struct {
	imageURL        string
	kernelURL       string
	commandLine     string
	commandLineArgs string
}

func newBootConfig(bc *domain.BootConfig, cfg *domain.Config) *bootConfig {
	cidr, _, _ := net.ParseCIDR(cfg.CIDR)
	metalCoreAddress := fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d", cidr.String(), cfg.Port)
	metalAPIURL := fmt.Sprintf("METAL_API_URL=%s://%s:%d%s", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort, cfg.ApiBasePath)

	cmdArgs := []string{metalCoreAddress, metalAPIURL}
	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		cmdArgs = append(cmdArgs, "DEBUG=1")
	}

	return &bootConfig{
		imageURL:        bc.MetalHammerImageURL,
		kernelURL:       bc.MetalHammerKernelURL,
		commandLine:     bc.MetalHammerCommandLine,
		commandLineArgs: strings.Join(cmdArgs, " "),
	}
}

func (h *endpointHandler) Boot(request *restful.Request, response *restful.Response) {
	mac := request.PathParameter("mac")

	h.log.Debug("request metal-api for a machine",
		zap.String("MAC", mac),
	)

	sc, machines := h.apiClient.FindMachines(mac)

	if sc == http.StatusOK {
		if len(machines) == 0 {
			rest.Respond(h.log, response, http.StatusOK, createBootResponse(h.apiClient, h.bootConfig, h.partitionID))
			return
		}
		if len(machines) == 1 {
			if machines[0].Allocation == nil {
				rest.Respond(h.log, response, http.StatusOK, createBootResponse(h.apiClient, h.bootConfig, h.partitionID))
				return
			}
			// Machine was already in the installation phase but crashed before finalizing allocation
			// we can boot into metal-hammer again.
			if !*machines[0].Allocation.Succeeded {
				rest.Respond(h.log, response, http.StatusOK, createBootResponse(h.apiClient, h.bootConfig, h.partitionID))
				return
			}
			h.log.Error("machine tries to pxe boot which is not expected.",
				zap.Int("statusCode", sc),
				zap.String("MAC", mac),
				zap.String("machineID", *machines[0].ID),
			)
			return
		}

		h.log.Error("more than one machines with same mac found, not booting machine.",
			zap.Int("statusCode", sc),
			zap.String("MAC", mac),
		)
		// FIXME this should not happen, we should consider returning a rec	overy image for digging into to root cause.
	} else {
		h.log.Error("request metal-api for a machine", zap.String("MAC", mac), zap.Int("statusCode", sc))
	}
}

func createBootResponse(apiClient domain.APIClient, cfg *bootConfig, partitionID string) domain.BootResponse {
	// try to update boot config
	s, err := apiClient.FindPartition(partitionID)
	if err == nil {
		cfg.imageURL = s.Bootconfig.Imageurl
		cfg.kernelURL = s.Bootconfig.Kernelurl
		cfg.commandLine = s.Bootconfig.Commandline
	}

	return domain.BootResponse{
		Kernel: cfg.kernelURL,
		InitRamDisk: []string{
			cfg.imageURL,
		},
		CommandLine: fmt.Sprintf("%s %s", cfg.commandLine, cfg.commandLineArgs),
	}
}
