package api

import (
	"fmt"
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) InstallImage(machineID string) (int, *models.V1MachineResponse) {
	params := machine.NewWaitForAllocationParams()
	params.ID = machineID

	ok, err := c.MachineClient.WaitForAllocation(params, c.Auth)
	if err != nil {
		switch e := err.(type) {
		case *machine.WaitForAllocationGatewayTimeout:
			zapup.MustRootLogger().Debug("Long polling timeout while GET from Metal-APIs wait endpoint",
				zap.String("machineID", machineID),
				zap.String("response", err.Error()),
			)
			return http.StatusNotModified, nil
		case *machine.WaitForAllocationDefault:
			zapup.MustRootLogger().Error("Failed to GET installation image from Metal-APIs wait endpoint",
				zap.String("machineID", machineID),
				zap.Error(fmt.Errorf(e.Error())),
			)
			return http.StatusInternalServerError, nil
		default:
			zapup.MustRootLogger().Error("Failed to GET installation image from Metal-APIs wait endpoint",
				zap.String("machineID", machineID),
				zap.Error(err),
			)
			return http.StatusInternalServerError, nil
		}
	}

	m := ok.Payload

	if m == nil || m.Allocation == nil || m.Allocation.Image == nil {
		zapup.MustRootLogger().Error("Got unexpected response from Metal-APIs wait endpoint",
			zap.String("machineID", machineID),
			zap.Any("machineWithToken", m),
		)
		return http.StatusExpectationFailed, nil
	}

	return http.StatusOK, m
}
