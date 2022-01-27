package event

import (
	"github.com/metal-stack/go-hal/pkg/api"
	"github.com/metal-stack/metal-core/internal/ipmi"
	"go.uber.org/zap"
)

func (h *eventHandler) UpdateBios(machineID, revision, description string, s3Cfg *api.S3Config) {
	h.Log.Info("update bios",
		zap.String("machine", machineID),
		zap.String("revision", revision),
		zap.String("description", description),
	)

	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.UpdateBios(h.Log, ipmiCfg, revision, s3Cfg)
	if err != nil {
		h.Log.Error("unable to update BIOS of machine",
			zap.String("machineID", machineID),
			zap.String("bios", revision),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) UpdateBmc(machineID, revision, description string, s3Cfg *api.S3Config) {
	h.Log.Info("update bmc",
		zap.String("machine", machineID),
		zap.String("revision", revision),
		zap.String("description", description),
	)

	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		h.Log.Error("unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.UpdateBmc(h.Log, ipmiCfg, revision, s3Cfg)
	if err != nil {
		h.Log.Error("unable to update BMC of machine",
			zap.String("machineID", machineID),
			zap.String("bmc", revision),
			zap.Error(err),
		)
	}
}
