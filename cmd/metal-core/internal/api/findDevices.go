package api

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"net/http"
)

func (c *apiClient) FindDevices(mac string) (int, []*models.MetalDevice) {
	params := device.NewSearchDeviceParams()
	params.Mac = &mac

	ok, err := c.DeviceClient.SearchDevice(params)
	if err != nil {
		zapup.MustRootLogger().Error("Device(s) not found",
			zap.String("mac", mac),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, ok.Payload
}
