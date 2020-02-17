package endpoint

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-lib/zapup"
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
