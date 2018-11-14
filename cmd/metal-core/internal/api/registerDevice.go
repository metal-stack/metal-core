package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/domain"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (c client) RegisterDevice(deviceId string, request *domain.MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice) {
	params := device.NewRegisterDeviceParams()
	params.Body = &models.ServiceRegisterRequest{
		UUID:   &deviceId,
		Siteid: &c.Config.SiteID,
		Rackid: &c.Config.RackID,
		Hardware: &models.MetalDeviceHardware{
			Memory:   request.Memory,
			CPUCores: request.CPUCores,
			Nics:     request.Nics,
			Disks:    request.Disks,
		},
		IPMI: request.IPMI,
	}

	ok, created, err := c.DeviceClient.RegisterDevice(params)
	if err != nil {
		zapup.MustRootLogger().Error("Failed to register device at Metal-API",
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	if ok != nil {
		return http.StatusOK, ok.Payload
	}
	return http.StatusOK, created.Payload
}
