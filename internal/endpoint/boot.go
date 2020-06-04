package endpoint

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (h *endpointHandler) Boot(request *restful.Request, response *restful.Response) {
	mac := request.PathParameter("mac")

	zapup.MustRootLogger().Debug("Request Metal-API for a machine",
		zap.String("MAC", mac),
	)

	sc, machines := h.APIClient().FindMachines(mac)
	if sc == http.StatusOK {
		bootResponse, err := createBootDiscoveryImageResponse(h)
		if err != nil {
			zapup.MustRootLogger().Error("Error assembling boot rsponse", zap.String("MAC", mac), zap.Error(err))
			return
		}

		if len(machines) == 0 {
			rest.Respond(response, http.StatusOK, bootResponse)
			return
		}
		if len(machines) == 1 {
			if machines[0].Allocation == nil {
				rest.Respond(response, http.StatusOK, bootResponse)
				return
			}
			// Machine was already in the installation phase but crashed before finalizing allocation
			// we can boot into metal-hammer again.
			if !*machines[0].Allocation.Succeeded {
				rest.Respond(response, http.StatusOK, bootResponse)
				return
			}
			zapup.MustRootLogger().Error("machine tries to pxe boot which is not expected.",
				zap.Int("statusCode", sc),
				zap.String("MAC", mac),
				zap.String("machineID", *machines[0].ID),
			)
			return
		}

		zapup.MustRootLogger().Error("more than one machines with same mac found, not booting machine.",
			zap.Int("statusCode", sc),
			zap.String("MAC", mac),
		)
		// FIXME this should not happen, we should consider returning a rec	overy image for digging into to root cause.
	} else {
		zapup.MustRootLogger().Error("Request Metal-API for a machine", zap.String("MAC", mac), zap.String("StatusCode", string(sc)))
	}
}

func createBootDiscoveryImageResponse(e *endpointHandler) (*domain.BootResponse, error) {
	cfg := e.Config

	cidr, _, _ := net.ParseCIDR(cfg.CIDR)
	metalCoreAddress := fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d", cidr.String(), cfg.Port)
	metalAPIURL := fmt.Sprintf("METAL_API_URL=%s://%s:%d%s", cfg.ApiProtocol, cfg.ApiIP, cfg.ApiPort, cfg.ApiBasePath)

	bc := e.BootConfig
	// try to update boot config
	s, err := e.APIClient().FindPartition(cfg.PartitionID)
	if err != nil {
		return nil, err
	}

	bc.MetalHammerImageURL = s.Bootconfig.Imageurl
	bc.MetalHammerKernelURL = s.Bootconfig.Kernelurl
	bc.MetalHammerCommandLine = s.Bootconfig.Commandline

	cmdline := []string{bc.MetalHammerCommandLine, metalCoreAddress, metalAPIURL}
	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		cmdline = append(cmdline, "DEBUG=1")
	}

	return &domain.BootResponse{
		Kernel: bc.MetalHammerKernelURL,
		InitRamDisk: []string{
			bc.MetalHammerImageURL,
		},
		CommandLine: strings.Join(cmdline, " "),
	}, err
}
