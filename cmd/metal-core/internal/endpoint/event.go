package endpoint

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

// ProvisioningEventType indicates an event emitted by a machine during the provisioning sequence
// FIXME factor out to metal-lib
type ProvisioningEventType string

// The enums for the machine provisioning events.
const (
	ProvisioningEventAlive            ProvisioningEventType = "Alive"
	ProvisioningEventCrashed          ProvisioningEventType = "Crashed"
	ProvisioningEventResetFailCount   ProvisioningEventType = "Reset Fail Count"
	ProvisioningEventPXEBooting       ProvisioningEventType = "PXE Booting"
	ProvisioningEventPlannedReboot    ProvisioningEventType = "Planned Reboot"
	ProvisioningEventPreparing        ProvisioningEventType = "Preparing"
	ProvisioningEventRegistering      ProvisioningEventType = "Registering"
	ProvisioningEventWaiting          ProvisioningEventType = "Waiting"
	ProvisioningEventInstalling       ProvisioningEventType = "Installing"
	ProvisioningEventBootingNewKernel ProvisioningEventType = "Booting New Kernel"
	ProvisioningEventPhonedHome       ProvisioningEventType = "Phoned Home"
)

func (h *endpointHandler) AddProvisioningEvent(request *restful.Request, response *restful.Response) {
	zapup.MustRootLogger().Info("event")

	event := &models.V1MachineProvisioningEvent{}
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
