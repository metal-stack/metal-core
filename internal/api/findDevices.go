package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (c client) FindDevices(mac string) (int, []*models.MetalDevice) {
	params := device.NewListDevicesParams()
	if ok, err := c.DeviceClient.ListDevices(params); err == nil {
		for _, dev := range ok.Payload {
			for _, nic := range dev.Hardware.Nics {
				if *nic.Mac == mac {
					return http.StatusOK, ok.Payload
				}
			}
		}
	}
	logging.Decorate(log.WithField("mac", mac)).
		Error("Device not found")
	return http.StatusNotFound, nil
}
