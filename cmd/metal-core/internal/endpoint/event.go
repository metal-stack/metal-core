package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"net/http"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func (h *endpointHandler) AddProvisioningEvent(request *restful.Request, response *restful.Response) {
	zapup.MustRootLogger().Info("event")

	var event *models.MetalProvisioningEvent
	err := request.ReadEntity(event)
	if err != nil {
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	machineID := request.PathParameter("id")
	zapup.MustRootLogger().Info("event", zap.String("machineID", machineID))

	err = h.APIClient().AddProvisioningEvent(machineID, event)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to send machine event back to api.",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(response, http.StatusOK, nil)
}
