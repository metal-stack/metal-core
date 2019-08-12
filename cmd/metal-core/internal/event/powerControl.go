package event

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) PowerOnMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOnMachine(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOffMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOffMachine(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerResetMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
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

func (h *eventHandler) PowerOnMachineLED(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOnMachineLED(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on machine chassis identify LED",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = h.APIClient().SetMachineLEDStateOn(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set machine chassis identify LED state to On",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOffMachineLED(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOffMachineLED(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off machine chassis identify LED",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = h.APIClient().SetMachineLEDStateOff(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set machine chassis identify LED state to Off",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}
