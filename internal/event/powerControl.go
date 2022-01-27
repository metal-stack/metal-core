package event

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"go.uber.org/zap"
)

func (h *eventHandler) PowerOnMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOnMachine(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("unable to power on machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOffMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOffMachine(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("unable to power off machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerResetMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerResetMachine(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("unable to power reset machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerCycleMachine(machineID string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerCycleMachine(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("unable to power cycle machine",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOnChassisIdentifyLED(machineID, description string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOnChassisIdentifyLED(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("unable to power on machine chassis identify LED",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = h.APIClient().SetChassisIdentifyLEDStateOn(machineID, description)
	if err != nil {
		h.Log.Error("unable to set machine chassis identify LED state to LED-ON",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) PowerOffChassisIdentifyLED(machineID, description string) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.PowerOffChassisIdentifyLED(h.Log, ipmiCfg)
	if err != nil {
		h.Log.Error("unable to power off machine chassis identify LED",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = h.APIClient().SetChassisIdentifyLEDStateOff(machineID, description)
	if err != nil {
		h.Log.Error("unable to set machine chassis identify LED state to LED-OFF",
			zap.String("machine", machineID),
			zap.Error(err),
		)
	}
}
