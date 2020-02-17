package api

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/pkg/errors"

	"go.uber.org/zap"
)

func (c *apiClient) AddProvisioningEvent(machineID string, event *models.V1MachineProvisioningEvent) error {
	zapup.MustRootLogger().Debug("event", zap.String("machineID", machineID))

	params := machine.NewAddProvisioningEventParams()
	params.ID = machineID
	params.Body = event
	_, err := c.MachineClient.AddProvisioningEvent(params, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to send machine event back to api.",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
		return errors.Wrapf(err, "unable to send event for machineID:%s with event:%s", machineID, *event.Event)
	}
	return nil
}
