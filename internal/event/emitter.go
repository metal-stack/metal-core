package event

import (
	"github.com/metal-stack/metal-core/internal/endpoint"
	"github.com/metal-stack/metal-core/internal/lldp"
	"github.com/metal-stack/metal-core/models"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
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

	zapup.MustRootLogger().Debug("Emit event",
		zap.String("type", t),
		zap.String("message", message),
	)

	return e.APIClient().AddProvisioningEvent(machineID, event)
}

func (e *Emitter) SendPhoneHomeEvent(msg *lldp.PhoneHomeMessage) {
	err := e.Emit(endpoint.ProvisioningEventPhonedHome, msg.MachineID, msg.Payload)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to phone home",
			zap.String("eventType", string(endpoint.ProvisioningEventPhonedHome)),
			zap.String("machineID", msg.MachineID),
			zap.String("payload", msg.Payload),
			zap.Error(err),
		)
	}
}