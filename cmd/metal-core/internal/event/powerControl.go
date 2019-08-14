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

func (h *eventHandler) PowerOnChassisIdentifyLED(machineID, description string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOnChassisIdentifyLED(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power on machine chassis identify LED",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = h.APIClient().SetChassisIdentifyLEDStateOn(machineID, description)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set machine chassis identify LED state to LED-ON",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOffChassisIdentifyLED(machineID, description string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOffChassisIdentifyLED(ipmiCfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to power off machine chassis identify LED",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = h.APIClient().SetChassisIdentifyLEDStateOff(machineID, description)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set machine chassis identify LED state to LED-OFF",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}
