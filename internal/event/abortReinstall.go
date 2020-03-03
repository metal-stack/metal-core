package event

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) AbortReinstallMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootDisk(ipmiCfg, h.DevMode)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to change boot order of machine",
			zap.String("machineID", machineID),
			zap.String("boot", "HD"),
			zap.Error(err),
		)
	}
}
