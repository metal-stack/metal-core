package event

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"go.uber.org/zap"
)

func (h *eventHandler) FreeMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootPXE(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("Unable to set boot order of machine to PXE",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	h.Log.Info("Freed machine",
		zap.String("machine", machineID),
	)

	err = ipmi.PowerCycleMachine(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("Unable to power cycle machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}
}
