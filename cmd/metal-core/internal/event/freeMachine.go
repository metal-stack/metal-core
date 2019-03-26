package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
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

	// This is our implementation of setBootDevice which has supermicro adoptions.
	err = ipmi.SetBootDevice(ipmiCfg, goipmi.BootDevicePxe)
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
