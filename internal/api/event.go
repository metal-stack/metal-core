package api

import (
	"time"

	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"

	"go.uber.org/zap"
)

func (c *apiClient) AddProvisioningEvent(machineID string, event *models.V1MachineProvisioningEvent) error {
	c.Log.Debug("event", zap.String("machineID", machineID))

	params := machine.NewAddProvisioningEventParams()
	params.ID = machineID
	params.Body = event
	params.WithTimeout(5 * time.Second)
	_, err := c.MachineClient.AddProvisioningEvent(params, c.Auth)
	if err != nil {
		c.Log.Error("unable to send provisioning event back to API",
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

	c.Log.Debug("emit event",
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

func (c *apiClient) PhoneHome(msg *phoneHomeMessage) {
	err := c.Emit(domain.ProvisioningEventPhonedHome, msg.machineID, msg.payload)
	if err != nil {
		c.Log.Error("unable to phone home",
			zap.String("eventType", string(domain.ProvisioningEventPhonedHome)),
			zap.String("machineID", msg.machineID),
			zap.String("message", msg.payload),
		)
	}
}
