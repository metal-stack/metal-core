package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) FreeDevice(device *models.MetalDevice) {
	var err error

	ipmiConn, err := h.APIClient().IPMIConfig(*device.ID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set read IPMI connection details",
			zap.Any("device", device),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootDevPxe(ipmiConn)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of device to HD",
			zap.Any("device", device),
			zap.Error(err),
		)
		return
	}

	zapup.MustRootLogger().Info("Freed device",
		zap.Any("device", device),
	)

	err = ipmi.PowerOff(ipmiConn)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off device",
			zap.Any("device", device),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOn(ipmiConn)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on device",
			zap.Any("device", device),
			zap.Error(err),
		)
		return
	}
}
