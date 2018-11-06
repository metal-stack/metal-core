package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
)

func (c client) RegisterDevice(deviceId string, request *domain.MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice) {
	params := device.NewRegisterDeviceParams()
	params.Body = &models.ServiceRegisterRequest{
		UUID:   &deviceId,
		Siteid: &c.Config().SiteID,
		Rackid: &c.Config().RackID,
		Hardware: &models.MetalDeviceHardware{
			Memory:   request.Memory,
			CPUCores: request.CPUCores,
			Nics:     request.Nics,
			Disks:    request.Disks,
		},
	}
	if ok, created, err := c.Device().RegisterDevice(params); err != nil {
		zapup.MustRootLogger().Error("Failed to register device at Metal-API",
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	} else if ok != nil {
		return http.StatusOK, ok.Payload
	} else {
		return http.StatusOK, created.Payload
	}
}
