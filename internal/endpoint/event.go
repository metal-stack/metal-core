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
		rest.Respond(h.Log, response, http.StatusInternalServerError, nil)
		return
	}

	machineID := request.PathParameter("id")
	h.Log.Debug("event", zap.String("machineID", machineID))

	err = h.APIClient().AddProvisioningEvent(machineID, event)
	if err != nil {
		rest.Respond(h.Log, response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(h.Log, response, http.StatusOK, nil)
}
