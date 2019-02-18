package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) FreeMachine(machine *models.MetalMachine) {
	var err error

	ipmiConn, err := h.APIClient().IPMIConfig(*machine.ID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set read IPMI connection details",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootMachinePXE(ipmiConn)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of machine to HD",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}

	zapup.MustRootLogger().Info("Freed machine",
		zap.Any("machine", machine),
	)

	err = ipmi.PowerOff(ipmiConn)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off machine",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOn(ipmiConn)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on machine",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}
}
