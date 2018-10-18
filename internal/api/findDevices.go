package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (c client) FindDevices(mac string) (int, []*models.MetalDevice) {
	params := device.NewSearchDeviceParams()
	params.Mac = &mac
	if ok, err := c.DeviceClient.SearchDevice(params); err != nil {
		logging.Decorate(log.WithField("mac", mac)).
			Error("Device(s) not found")
		return http.StatusInternalServerError, nil
	} else {
		return http.StatusOK, ok.Payload
	}
}
