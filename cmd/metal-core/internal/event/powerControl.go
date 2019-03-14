package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) PowerOnMachine(machine *models.MetalMachine, params []string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(*machine.ID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOn(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on machine",
			zap.Any("machine", machine),
			zap.Strings("params", params),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOffMachine(machine *models.MetalMachine, params []string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(*machine.ID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOff(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off machine",
			zap.Any("machine", machine),
			zap.Strings("params", params),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerResetMachine(machine *models.MetalMachine, params []string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(*machine.ID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machine),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerReset(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power reset machine",
			zap.Any("machine", machine),
			zap.Strings("params", params),
			zap.Error(err),
		)
	}
}
