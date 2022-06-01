package endpoint

import (
	"net/http"

	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"

	"github.com/metal-stack/metal-core/internal/rest"
)

func (h *endpointHandler) Register(request *restful.Request, response *restful.Response) {
	req := &domain.MetalHammerRegisterMachineRequest{}
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

	h.log.Debug("register machine at metal-api",
		zap.String("machineID", machineID),
		zap.String("IPMI-Address", req.IPMIAddress()),
		zap.String("IPMI-Interface", req.IPMIInterface()),
		zap.String("IPMI-MAC", req.IPMIMAC()),
		zap.String("IPMI-User", req.IPMIUser()),
	)

	sc, machine := h.apiClient.RegisterMachine(machineID, req)

	if sc != http.StatusOK {
		errMsg := "failed to register machine"
		h.log.Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("machineID", machineID),
			zap.Any("machine", machine),
			zap.Error(err),
		)
		rest.RespondError(h.log, response, http.StatusInternalServerError, errMsg)
		return
	}

	h.log.Info("machine registered",
		zap.Int("statusCode", sc),
		zap.Any("machine", machine),
	)

	rest.Respond(h.log, response, http.StatusOK, machine)
}
