package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

func (c client) RegisterDevice(deviceId string, request *domain.MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice) {
	params := device.NewRegisterDeviceParams()
	params.Body = &models.ServiceRegisterRequest{
		UUID:   &deviceId,
		Siteid: &c.GetConfig().SiteID,
		Rackid: &c.GetConfig().RackID,
		Hardware: &models.MetalDeviceHardware{
			Memory:   request.Memory,
			CPUCores: request.CPUCores,
			Nics:     request.Nics,
			Disks:    request.Disks,
		},
	}
	if ok, created, err := c.DeviceClient.RegisterDevice(params); err != nil {
		logging.Decorate(log.WithFields(log.Fields{})).
			Error("Failed to register device at Metal-API")
		return http.StatusInternalServerError, nil
	} else if ok != nil {
		return http.StatusOK, ok.Payload
	} else {
		return http.StatusOK, created.Payload
	}
}
