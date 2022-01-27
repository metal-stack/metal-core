package event

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"go.uber.org/zap"
)

func (h *eventHandler) ReinstallMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootPXE(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("unable to change boot order of machine",
			zap.String("machineID", machineID),
			zap.String("boot", "PXE"),
			zap.Error(err),
		)
	}

	err = ipmi.PowerResetMachine(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("unable to power reset machine",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
	}
}
