package event

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) ReinstallMachine(machineID string) {
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
		zapup.MustRootLogger().Error("Unable to change boot order of machine",
			zap.String("machineID", machineID),
			zap.String("boot", "PXE"),
			zap.Error(err),
		)
	}

	err = ipmi.PowerResetMachine(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power reset machine",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
	}
}
