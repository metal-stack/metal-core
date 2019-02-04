package api

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (c *apiClient) RegisterDevice(deviceID string, request *domain.MetalHammerRegisterDeviceRequest) (int, *models.MetalDevice) {
	siteId := c.Config.SiteID
	rackId := c.Config.RackID
	params := device.NewRegisterDeviceParams()
	params.Body = &models.MetalRegisterDevice{
		UUID:   &deviceID,
		Siteid: &siteId,
		Rackid: &rackId,
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
			zap.String("deviceID", deviceID),
			zap.String("siteID", siteId),
			zap.String("rackID", rackId),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	if ok != nil {
		return http.StatusOK, ok.Payload
	}
	return http.StatusOK, created.Payload
}
