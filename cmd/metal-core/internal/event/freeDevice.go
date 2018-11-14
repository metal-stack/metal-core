package event

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (l listener) FreeDevice(device *models.MetalDevice) {
	var err error

	err = ipmi.SetBootDevPxe(l.IpmiConnection)
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

	err = ipmi.PowerOff(l.IpmiConnection)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off device",
			zap.Any("device", device),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOn(l.IpmiConnection)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on device",
			zap.Any("device", device),
			zap.Error(err),
		)
		return
	}
}
