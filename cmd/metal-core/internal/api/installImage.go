package api

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (c client) InstallImage(deviceId string) (int, *models.MetalDeviceWithPhoneHomeToken) {
	params := device.NewWaitForAllocationParams()
	params.ID = deviceId

	ok, err := c.DeviceClient.WaitForAllocation(params)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to GET installation image from Metal-APIs wait endpoint",
			zap.String("deviceID", deviceId),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, ok.Payload
}
