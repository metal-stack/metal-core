package event

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"go.uber.org/zap"
)

func (h *eventHandler) FreeMachine(machineID string) {
	ipmiCfg, err := h.apiClient.IPMIConfig(machineID)
	if err != nil {
		h.log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootPXE(h.log, ipmiCfg)
	if err != nil {
		h.log.Error("unable to set boot order of machine to PXE",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	h.log.Info("freed machine",
		zap.String("machine", machineID),
	)

	err = ipmi.PowerCycleMachine(h.log, ipmiCfg)
	if err != nil {
		h.log.Error("unable to power cycle machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}
}
