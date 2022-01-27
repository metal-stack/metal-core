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
		h.Log.Error("cannot read request",
			zap.Error(err),
		)
		rest.RespondError(h.Log, response, http.StatusBadRequest, errMsg)
		return
	}

	machineID := request.PathParameter("id")

	h.Log.Debug("abort reinstall",
		zap.String("machineID", machineID),
		zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
	)

	sc, bootInfo := h.APIClient().AbortReinstall(machineID, req)
	if sc != http.StatusOK {
		errMsg := "failed to abort reinstall"
		h.Log.Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("machineID", machineID),
			zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
			zap.Any("boot information", bootInfo),
			zap.Error(err),
		)
		rest.Respond(h.Log, response, http.StatusInternalServerError, errMsg)
		return
	}

	if h.Config.ChangeBootOrder {
		ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
		if err != nil {
			rest.Respond(h.Log, response, http.StatusInternalServerError, err)
			return
		}

		err = ipmi.SetBootDisk(h.Log, ipmiCfg)
		if err != nil {
			h.Log.Error("unable to set boot order of machine to HD",
				zap.String("machineID", machineID),
				zap.Any("boot information", bootInfo),
				zap.Error(err),
			)
			rest.Respond(h.Log, response, http.StatusInternalServerError, err)
			return
		}
	}

	h.Log.Info("machine reinstall aborted",
		zap.Int("statusCode", sc),
		zap.String("machineID", machineID),
		zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
		zap.Any("boot information", bootInfo),
	)

	rest.Respond(h.Log, response, http.StatusOK, bootInfo)
}
