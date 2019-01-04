package api

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (c client) InstallImage(deviceID string) (int, *models.MetalDeviceWithPhoneHomeToken) {
	params := device.NewWaitForAllocationParams()
	params.ID = deviceID

	ok, err := c.DeviceClient.WaitForAllocation(params)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to GET installation image from Metal-APIs wait endpoint",
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, ok.Payload
}
