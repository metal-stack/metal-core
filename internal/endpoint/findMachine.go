package endpoint

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-core/internal/rest"
	"go.uber.org/zap"
)

func (h *endpointHandler) FindMachine(request *restful.Request, response *restful.Response) {
	machineID := request.PathParameter("id")

	h.Log.Debug("Request Metal-API to find a machine",
		zap.String("machineID", machineID),
	)

	machine, err := h.APIClient().FindMachine(machineID)
	if err != nil {
		errMsg := "Failed to find machine"
		h.Log.Error(errMsg,
			zap.String("machineID", machineID),
		)
		rest.Respond(h.Log, response, http.StatusInternalServerError, errMsg)
	}

	rest.Respond(h.Log, response, http.StatusOK, machine)
}
