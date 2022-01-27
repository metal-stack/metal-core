package api

import (
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"go.uber.org/zap"
)

func (c *apiClient) FindMachine(id string) (*models.V1MachineResponse, error) {
	findMachine := machine.NewFindMachineParams()
	findMachine.ID = id
	ok, err := c.MachineClient.FindMachine(findMachine, c.Auth)
	if err != nil {
		c.Log.Error("machine not found",
			zap.String("ID", id),
			zap.Error(err),
		)
		return nil, err
	}
	return ok.Payload, nil
}
