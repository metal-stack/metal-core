package endpoint

import (
	"net/http"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/metal-stack/metal-core/internal/rest"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"
)

type emptyBootRepsonse struct{}

func (h *endpointHandler) Dhcp(request *restful.Request, response *restful.Response) {
	machineID := request.PathParameter("id")

	h.Log.Debug("emit pxe boot event from machine",
		zap.String("machineID", machineID),
	)

	eventType := string(domain.ProvisioningEventPXEBooting)
	event := &v1.EventServiceSendRequest{Events: map[string]*v1.MachineProvisioningEvent{
		machineID: {
			Event:   eventType,
			Message: "machine sent extended dhcp request",
		},
	}}
	_, err := h.APIClient().Send(event)
	if err != nil {
		h.Log.Debug("dhcp: unable to emit PXEBooting provisioning event... ignoring",
			zap.String("machineID", machineID),
			zap.String("error", err.Error()),
		)
	}

	// the response of the extended dhcp request does not need to contain useful information
	// only the ipxe http request following the dhcp extended request will need to contain the boot image data
	rest.Respond(h.Log, response, http.StatusOK, emptyBootRepsonse{})
}
