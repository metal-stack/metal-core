package api

import (
	"github.com/metal-stack/metal-core/internal/lldp"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/zapup"

	"go.uber.org/zap"
)

func (c *apiClient) AddProvisioningEvent(machineID string, event *models.V1MachineProvisioningEvent) error {
	zapup.MustRootLogger().Debug("event", zap.String("machineID", machineID))

	params := machine.NewAddProvisioningEventParams()
	params.ID = machineID
	params.Body = event
	_, err := c.MachineClient.AddProvisioningEvent(params, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to send provisioning event back to API",
			zap.String("eventType", *event.Event),
			zap.String("machineID", machineID),
			zap.String("message", event.Message),
			zap.Error(err),
		)
	}
	return err
}

func (c *apiClient) Emit(eventType domain.ProvisioningEventType, machineID, message string) error {
	et := string(eventType)

	zapup.MustRootLogger().Debug("Emit event",
		zap.String("eventType", et),
		zap.String("machineID", machineID),
		zap.String("message", message),
	)

	event := &models.V1MachineProvisioningEvent{
		Event:   &et,
		Message: message,
	}
	return c.AddProvisioningEvent(machineID, event)
}

func (c *apiClient) PhoneHome(msg *lldp.PhoneHomeMessage) {
	err := c.Emit(domain.ProvisioningEventPhonedHome, msg.MachineID, msg.Payload)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to phone home",
			zap.String("eventType", string(domain.ProvisioningEventPhonedHome)),
			zap.String("machineID", msg.MachineID),
			zap.String("message", msg.Payload),
		)
	}
}
