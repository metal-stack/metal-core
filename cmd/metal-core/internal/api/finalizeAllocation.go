package api

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) FinalizeAllocation(machineid, consolepassword string) (*machine.FinalizeAllocationOK, error) {
	body := &models.V1MachineFinalizeAllocationRequest{
		ConsolePassword: &consolepassword,
	}
	params := machine.NewFinalizeAllocationParams()
	params.ID = machineid
	params.Body = body

	ok, err := c.MachineClient.FinalizeAllocation(params, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("Finalize failed",
			zap.String("machineid", machineid),
			zap.Error(err),
		)
	}
	return ok, err
}
