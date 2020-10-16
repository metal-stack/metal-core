package endpoint

import (
	"github.com/metal-stack/metal-core/pkg/domain"
	"net/http"

	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-go/api/models"

	"github.com/emicklei/go-restful"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

type emptyBootRepsonse struct{}

func (h *endpointHandler) Dhcp(request *restful.Request, response *restful.Response) {
	machineID := request.PathParameter("id")

	zapup.MustRootLogger().Debug("emit pxe boot event from machine",
		zap.String("machineID", machineID),
	)

	eventType := string(domain.ProvisioningEventPXEBooting)
	event := &models.V1MachineProvisioningEvent{
		Event:   &eventType,
		Message: "machine sent extended dhcp request",
	}
	err := h.APIClient().AddProvisioningEvent(machineID, event)
	if err != nil {
		zapup.MustRootLogger().Debug("dhcp: unable to emit PXEBooting provisioning event... ignoring",
			zap.String("machineID", machineID),
			zap.String("error", err.Error()),
		)
	}

	// the response of the extended dhcp request does not need to contain useful information
	// only the ipxe http request following the dhcp extended request will need to contain the boot image data
	rest.Respond(response, http.StatusOK, emptyBootRepsonse{})
}
