package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (c client) FindDevice(mac string) (int, *models.MetalDevice) {
	params := device.NewFindDeviceParams()
	params.ID = mac
	if ok, err := c.DeviceClient.FindDevice(params); err != nil {
		logging.Decorate(log.WithField("mac", mac)).
			Error("Device not found")
		return http.StatusNotFound, nil
	} else {
		return http.StatusOK, ok.Payload
	}
}
