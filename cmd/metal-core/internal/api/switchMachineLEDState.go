package api

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) SetMachineLEDStateOn(machineID, description string) error {
	return c.setMachineLEDState(machineID, description, "On")
}

func (c *apiClient) SetMachineLEDStateOff(machineID, description string) error {
	return c.setMachineLEDState(machineID, description, "Off")
}

func (c *apiClient) setMachineLEDState(machineID, description, state string) error {
	params := machine.NewSetMachineLEDStateParams()
	params.SetID(machineID)
	req := &models.V1MachineLEDState{
		Value:       &state,
		Description: &description,
	}
	params.SetBody(req)

	ok, err := c.MachineClient.SetMachineLEDState(params, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Cannot set machine chassis identify LED state",
			zap.String("machineID", machineID),
			zap.String("state", state),
			zap.Error(err),
		)
		return err
	}

	if ok.Payload != nil && ok.Payload.Ledstate != nil {
		zapup.MustRootLogger().Info("Set machine chassis identify LED state",
			zap.String("machineID", machineID),
			zap.String("state", *ok.Payload.Ledstate.Value),
			zap.String("description", *ok.Payload.Ledstate.Description),
		)
	}

	return nil
}
