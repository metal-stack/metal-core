package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/log"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"go.uber.org/zap"
	"net/http"
)

func (c client) InstallImage(deviceId string) (int, *models.MetalDevice) {
	params := device.NewWaitForAllocationParams()
	params.ID = deviceId
	if ok, err := c.Device().WaitForAllocation(params); err != nil {
		log.Get().Error("Failed to GET installation image from Metal-APIs wait endpoint",
			zap.String("deviceID", deviceId),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	} else {
		return http.StatusOK, ok.Payload
	}
}
