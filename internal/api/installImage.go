package api

import (
	"fmt"
	"net/http"

	"github.com/metal-stack/metal-core/client/machine"
	"github.com/metal-stack/metal-core/models"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) InstallImage(machineID string) (int, *models.V1MachineResponse) {
	params := machine.NewWaitForAllocationParams()
	params.ID = machineID

	ok, err := c.MachineClient.WaitForAllocation(params, c.Auth)
	if err != nil {
		response := ""
		statusCode := int32(0)
		switch e := err.(type) {
		case *machine.WaitForAllocationGatewayTimeout:
			if e.Payload != nil && e.Payload.Statuscode != nil {
				response = *e.Payload.Message
				statusCode = *e.Payload.Statuscode
			}
			zapup.MustRootLogger().Debug("Long polling timeout while GET from Metal-APIs wait endpoint",
				zap.String("machineID", machineID),
				zap.Int32("statusCode", statusCode),
				zap.String("response", response),
			)
			return http.StatusNotModified, nil
		case *machine.WaitForAllocationDefault:
			if e.Payload != nil && e.Payload.Statuscode != nil {
				response = *e.Payload.Message
				statusCode = *e.Payload.Statuscode
			}
			zapup.MustRootLogger().Error("Failed to GET installation image from Metal-APIs wait endpoint",
				zap.String("machineID", machineID),
				zap.Int32("statusCode", statusCode),
				zap.Error(fmt.Errorf(response)),
			)
			return http.StatusInternalServerError, nil
		default:
			zapup.MustRootLogger().Debug("Metal-APIs wait for installation image timeout",
				zap.String("machineID", machineID),
				zap.String("response", e.Error()),
			)
			return http.StatusNotModified, nil
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
