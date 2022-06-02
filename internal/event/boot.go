package event

import (
	"github.com/metal-stack/metal-core/internal/ipmi"
	"go.uber.org/zap"
)

func (h *eventHandler) PowerBootBiosMachine(machineID string) {
	ipmiCfg, err := h.apiClient.IPMIConfig(machineID)
	if err != nil {
		h.log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootBios(h.log, ipmiCfg)
	if err != nil {
		h.log.Error("unable to set boot order of machine to BIOS",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}
}

func (h *eventHandler) PowerBootDiskMachine(machineID string) {
	ipmiCfg, err := h.apiClient.IPMIConfig(machineID)
	if err != nil {
		h.log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.SetBootDisk(h.log, ipmiCfg)
	if err != nil {
		h.log.Error("unable to set boot order of machine to disk",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}
}

func (h *eventHandler) PowerBootPxeMachine(machineID string) {
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
}
