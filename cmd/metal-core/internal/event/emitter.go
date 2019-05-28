package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

type Emitter struct {
	*domain.AppContext
}

func NewEmitter(appContext *domain.AppContext) *Emitter {
	return &Emitter{
		AppContext: appContext,
	}
}

func (e *Emitter) Emit(eventType endpoint.ProvisioningEventType, machineID, message string) error {
	t := string(eventType)
	event := &models.V1MachineProvisioningEvent{
		Event:   &t,
		Message: message,
	}

	zapup.MustRootLogger().Info("Emit event",
		zap.String("type", t),
		zap.String("message", message),
	)

	return e.APIClient().AddProvisioningEvent(machineID, event)
}
