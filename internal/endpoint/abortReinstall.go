package endpoint

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-core/pkg/domain"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-core/internal/rest"
)

func (h *endpointHandler) AbortReinstall(request *restful.Request, response *restful.Response) {
	req := &domain.MetalHammerAbortReinstallRequest{}
	err := request.ReadEntity(req)
	if err != nil {
		errMsg := "Unable to read body"
		zapup.MustRootLogger().Error("Cannot read request",
			zap.Error(err),
		)
		rest.RespondError(response, http.StatusBadRequest, errMsg)
		return
	}

	machineID := request.PathParameter("id")

	zapup.MustRootLogger().Debug("Abort reinstall",
		zap.String("machineID", machineID),
		zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
	)

	sc, bootInfo := h.APIClient().AbortReinstall(machineID, req)
	if sc != http.StatusOK {
		errMsg := "Failed to abort reinstall"
		zapup.MustRootLogger().Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("machineID", machineID),
			zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
			zap.Any("boot information", bootInfo),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, errMsg)
		return
	}

	if h.Config.ChangeBootOrder {
		ipmiCfg, err := h.APIClient().IPMIConfig(machineID, h.Compliance)
		if err != nil {
			rest.Respond(response, http.StatusInternalServerError, err)
			return
		}

		err = ipmi.SetBootDisk(ipmiCfg)
		if err != nil {
			zapup.MustRootLogger().Error("Unable to set boot order of machine to HD",
				zap.String("machineID", machineID),
				zap.Any("boot information", bootInfo),
				zap.Error(err),
			)
			rest.Respond(response, http.StatusInternalServerError, err)
			return
		}
	}

	zapup.MustRootLogger().Info("Machine reinstall aborted",
		zap.Int("statusCode", sc),
		zap.String("machineID", machineID),
		zap.Bool("primary disk already wiped", req.PrimaryDiskWiped),
		zap.Any("boot information", bootInfo),
	)

	rest.Respond(response, http.StatusOK, bootInfo)
}
