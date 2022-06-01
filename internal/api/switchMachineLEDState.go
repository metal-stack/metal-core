package api

import (
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
)

const (
	ledStateOn  = "LED-ON"
	ledStateOff = "LED-OFF"
)

func (c *apiClient) SetChassisIdentifyLEDStateOn(machineID, description string) error {
	return c.setChassisIdentifyLEDState(machineID, description, ledStateOn)
}

func (c *apiClient) SetChassisIdentifyLEDStateOff(machineID, description string) error {
	return c.setChassisIdentifyLEDState(machineID, description, ledStateOff)
}

func (c *apiClient) setChassisIdentifyLEDState(machineID, description, state string) error {
	params := machine.NewSetChassisIdentifyLEDStateParams()
	params.SetID(machineID)
	req := &models.V1ChassisIdentifyLEDState{
		Value:       &state,
		Description: &description,
	}
	params.SetBody(req)

	ok, err := c.machineClient.SetChassisIdentifyLEDState(params, c.auth)
	if err != nil {
		return err
	}

	if ok.Payload != nil && ok.Payload.Ledstate != nil {
		c.log.Info("set machine chassis identify led state",
			zap.String("machineID", machineID),
			zap.String("state", *ok.Payload.Ledstate.Value),
			zap.String("description", *ok.Payload.Ledstate.Description),
		)
	}

	return nil
}
