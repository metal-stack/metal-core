package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"net/http"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
)

func (h *endpointHandler) Register(request *restful.Request, response *restful.Response) {
	req := &domain.MetalHammerRegisterMachineRequest{}

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

	zapup.MustRootLogger().Info("Register machine at Metal-API",
		zap.String("machineID", machineID),
		zap.String("IPMI-Address", req.IPMIAddress()),
		zap.String("IPMI-Interface", req.IPMIInterface()),
		zap.String("IPMI-MAC", req.IPMIMAC()),
		zap.String("IPMI-User", req.IPMIUser()),
	)

	sc, machine := h.APIClient().RegisterMachine(machineID, req)

	if sc != http.StatusOK {
		errMsg := "Failed to register machine"
		zapup.MustRootLogger().Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("machineID", machineID),
			zap.Any("machine", machine),
			zap.Error(err),
		)
		rest.RespondError(response, http.StatusInternalServerError, errMsg)
		return
	}

	zapup.MustRootLogger().Info("Machine registered",
		zap.Int("statusCode", sc),
		zap.Any("machine", machine),
	)
	rest.Respond(response, http.StatusOK, machine)
}
