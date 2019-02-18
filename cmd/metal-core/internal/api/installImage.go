package api

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) InstallImage(machineID string) (int, *models.MetalMachineWithPhoneHomeToken) {
	params := machine.NewWaitForAllocationParams()
	params.ID = machineID

	ok, err := c.MachineClient.WaitForAllocation(params)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to GET installation image from Metal-APIs wait endpoint",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, ok.Payload
}
