package event

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) BootBiosMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootBios(ipmiCfg, h.DevMode)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of machine to BIOS",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerResetMachine(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power reset machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}
