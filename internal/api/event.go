package api

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"

	"go.uber.org/zap"
)

func (c *apiClient) AddProvisioningEvent(machineID string, event *models.V1MachineProvisioningEvent) error {
	c.log.Debug("event", zap.String("machineID", machineID))

	params := machine.NewAddProvisioningEventParams()
	params.ID = machineID
	params.Body = event
	params.WithTimeout(5 * time.Second)
	_, err := c.machineClient.AddProvisioningEvent(params, c.auth)
	if err != nil {
		c.log.Error("unable to send provisioning event back to API",
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

	c.log.Debug("emit event",
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

func (c *apiClient) PhoneHome(msgs []phoneHomeMessage) {
	c.log.Debug("phonehome",
		zap.String("machines", fmt.Sprintf("%v", msgs)),
	)
	c.log.Info("phonehome",
		zap.Int("machines", len(msgs)),
	)
	events := models.V1MachineProvisioningEvents{}
	phonedHomeEvent := string(domain.ProvisioningEventPhonedHome)
	for i := range msgs {
		msg := msgs[i]
		event := models.V1MachineProvisioningEvent{
			Event:   &phonedHomeEvent,
			Message: msg.payload,
			Time:    strfmt.DateTime(msg.time),
		}
		events[msg.machineID] = event
	}

	params := machine.NewAddProvisioningEventsParams()
	params.Body = events
	params.WithTimeout(5 * time.Second)
	resp, err := c.machineClient.AddProvisioningEvents(params, c.auth)
	if err != nil {
		c.log.Error("unable to send provisioning event back to API",
			zap.Error(err),
		)
	}
	if resp != nil && resp.Payload != nil && resp.Payload.Events != nil {
		c.log.Info("phonehome sent",
			zap.Int64("machines", *resp.Payload.Events),
		)
	}
}
