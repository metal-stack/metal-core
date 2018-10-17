package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (c client) InstallImage(deviceId string) (int, *models.MetalDevice) {
	params := device.NewWaitForAllocationParams()
	params.ID = deviceId
	if ok, err := c.DeviceClient.WaitForAllocation(params); err != nil {
		logging.Decorate(log.WithField("deviceID", deviceId)).
			Error("Failed to GET installation image from Metal-APIs wait endpoint")
		return http.StatusInternalServerError, nil
	} else {
		return http.StatusOK, ok.Payload
	}
}
