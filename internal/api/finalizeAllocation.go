package api

import (
	"github.com/metal-stack/metal-core/client/machine"
	"github.com/metal-stack/metal-core/models"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) FinalizeAllocation(machineID, consolePassword, primaryDisk, osPartition string) (*machine.FinalizeAllocationOK, error) {
	body := &models.V1MachineFinalizeAllocationRequest{
		ConsolePassword: &consolePassword,
		Primarydisk:     &primaryDisk,
		Ospartition:     &osPartition,
	}
	params := machine.NewFinalizeAllocationParams()
	params.ID = machineID
	params.Body = body

	ok, err := c.MachineClient.FinalizeAllocation(params, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Finalize failed",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
	}
	return ok, err
}
