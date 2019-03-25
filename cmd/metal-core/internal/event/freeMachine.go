package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) FreeMachine(machine *models.MetalMachine) {
	ipmiCfg, err := h.APIClient().IPMIConfig(*machine.ID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootMachinePXE(ipmiCfg)
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

	err = ipmi.PowerCycle(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power cycle machine",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}
}
