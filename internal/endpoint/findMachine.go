package endpoint

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-core/internal/rest"
	"go.uber.org/zap"
)

func (h *endpointHandler) FindMachine(request *restful.Request, response *restful.Response) {
	machineID := request.PathParameter("id")

	h.log.Debug("request metal-api to Find a machine",
		zap.String("machineID", machineID),
	)

	machine, err := h.apiClient.FindMachine(machineID)
	if err != nil {
		errMsg := "failed to Find machine"
		h.log.Error(errMsg,
			zap.String("machineID", machineID),
		)
		rest.Respond(h.log, response, http.StatusInternalServerError, errMsg)
	}

	rest.Respond(h.log, response, http.StatusOK, machine)
}
