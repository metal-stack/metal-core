package api

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"net/http"
)

func (c client) FindDevices(mac string) (int, []*models.MetalDevice) {
	params := device.NewSearchDeviceParams()
	params.Mac = &mac
	if ok, err := c.Device().SearchDevice(params); err != nil {
		zapup.MustRootLogger().Error("Device(s) not found",
			zap.String("mac", mac),
			zap.Error(err),
		)
		return http.StatusInternalServerError, nil
	} else {
		return http.StatusOK, ok.Payload
	}
}
