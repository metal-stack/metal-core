package endpoint

import (
	"net/http"

	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-go/api/models"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"
)

func (h *endpointHandler) AddProvisioningEvent(request *restful.Request, response *restful.Response) {
	event := &models.V1MachineProvisioningEvent{}
	err := request.ReadEntity(event)
	if err != nil {
		rest.Respond(h.log, response, http.StatusInternalServerError, nil)
		return
	}

	machineID := request.PathParameter("id")
	h.log.Debug("event", zap.String("machineID", machineID))

	err = h.apiClient.AddProvisioningEvent(machineID, event)
	if err != nil {
		rest.Respond(h.log, response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(h.log, response, http.StatusOK, nil)
}
