package endpoint

import (
	"net/http"

	"github.com/metal-stack/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-core/internal/rest"
)

func (h *endpointHandler) AbortReinstall(request *restful.Request, response *restful.Response) {
	req := &domain.MetalHammerAbortReinstallRequest{}
	err := request.ReadEntity(req)
	if err != nil {
		errMsg := "Unable to read body"
		h.log.Error("cannot read request",
			zap.Error(err),
		)
		rest.RespondError(h.log, response, http.StatusBadRequest, errMsg)
		return
	}

	machineID := request.PathParameter("id")

	h.log.Debug("abort reinstall",
		zap.String("machineID", machineID),
		zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
	)

	sc, bootInfo := h.apiClient.AbortReinstall(machineID, req)
	if sc != http.StatusOK {
		errMsg := "failed to abort reinstall"
		h.log.Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("machineID", machineID),
			zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
			zap.Any("boot information", bootInfo),
			zap.Error(err),
		)
		rest.Respond(h.log, response, http.StatusInternalServerError, errMsg)
		return
	}

	if h.changeBootOrder {
		ipmiCfg, err := h.apiClient.IPMIConfig(machineID)
		if err != nil {
			rest.Respond(h.log, response, http.StatusInternalServerError, err)
			return
		}

		err = ipmi.SetBootDisk(h.log, ipmiCfg)
		if err != nil {
			h.log.Error("unable to set boot order of machine to HD",
				zap.String("machineID", machineID),
				zap.Any("boot information", bootInfo),
				zap.Error(err),
			)
			rest.Respond(h.log, response, http.StatusInternalServerError, err)
			return
		}
	}

	h.log.Info("machine reinstall aborted",
		zap.Int("statusCode", sc),
		zap.String("machineID", machineID),
		zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
		zap.Any("boot information", bootInfo),
	)

	rest.Respond(h.log, response, http.StatusOK, bootInfo)
}
