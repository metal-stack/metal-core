package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) PowerOnMachine(machineID string, params []string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOnMachine(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on machine",
			zap.Any("machine", machineID),
			zap.Strings("params", params),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOffMachine(machineID string, params []string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOffMachine(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off machine",
			zap.Any("machine", machineID),
			zap.Strings("params", params),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerResetMachine(machineID string, params []string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerResetMachine(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power reset machine",
			zap.Any("machine", machineID),
			zap.Strings("params", params),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOnMachineLED(machineID string, params []string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOnMachineLED(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on machine LED",
			zap.Any("machine", machineID),
			zap.Strings("params", params),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOffMachineLED(machineID string, params []string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.Any("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOffMachineLED(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off machine LED",
			zap.Any("machine", machineID),
			zap.Strings("params", params),
			zap.Error(err),
		)
	}
}
