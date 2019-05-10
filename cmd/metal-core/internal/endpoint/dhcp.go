package endpoint

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

type emptyBootRepsonse struct{}

func (e *endpointHandler) Dhcp(request *restful.Request, response *restful.Response) {
	guid := request.PathParameter("id")

	zapup.MustRootLogger().Info("emit pxe boot event from machine",
		zap.String("guid", guid),
	)

	eventType := string(ProvisioningEventPXEBooting)
	event := &models.V1MachineProvisioningEvent{
		Event:   &eventType,
		Message: "machine sent extended dhcp request",
	}
	err := e.APIClient().AddProvisioningEvent(guid, event)
	if err != nil {
		zapup.MustRootLogger().Error("request metal-api event endpoint for machine", zap.String("guid", guid), zap.String("error", err.Error()))
	}

	// the response of the extended dhcp request does not need to contain useful information
	// only the ipxe http request following the dhcp extended request will need to contain the boot image data
	rest.Respond(response, http.StatusOK, emptyBootRepsonse{})
}
