package api

import (
	"github.com/metal-stack/metal-core/client/machine"
	"github.com/metal-stack/metal-core/models"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) FindMachine(id string) (*models.V1MachineResponse, error) {
	findMachine := machine.NewFindMachineParams()
	findMachine.ID = id
	ok, err := c.MachineClient.FindMachine(findMachine, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Machine not found",
			zap.String("ID", id),
			zap.Error(err),
		)
		return nil, err
	}
	return ok.Payload, nil
}
