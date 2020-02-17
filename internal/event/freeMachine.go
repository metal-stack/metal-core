package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) FreeMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootPXE(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of machine to HD",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	zapup.MustRootLogger().Info("Freed machine",
		zap.String("machine", machineID),
	)

	err = ipmi.PowerCycleMachine(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power cycle machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}
}
