package endpoint

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func (h *endpointHandler) FindMachine(request *restful.Request, response *restful.Response) {
	machineID := request.PathParameter("id")

	zapup.MustRootLogger().Debug("Request Metal-API to find a machine",
		zap.String("machineID", machineID),
	)

	machine, err := h.APIClient().FindMachine(machineID)
	if err != nil {
		errMsg := "Failed to find machine"
		zapup.MustRootLogger().Error(errMsg,
			zap.String("machineID", machineID),
		)
		rest.Respond(response, http.StatusInternalServerError, errMsg)
	}

	rest.Respond(response, http.StatusOK, machine)
}
