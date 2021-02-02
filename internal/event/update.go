package event

import (
	"github.com/metal-stack/go-hal/pkg/api"
	"github.com/metal-stack/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func (h *eventHandler) UpdateBios(machineID, revision string, s3Cfg *api.S3Config) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.UpdateBios(ipmiCfg, revision, s3Cfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to change boot order of machine",
			zap.String("machineID", machineID),
			zap.String("boot", "PXE"),
			zap.Error(err),
		)
	}
}

func (h *eventHandler) UpdateBmc(machineID, revision string, s3Cfg *api.S3Config) {
	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to read IPMI connection details",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return
	}

	err = ipmi.UpdateBmc(ipmiCfg, revision, s3Cfg)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to change boot order of machine",
			zap.String("machineID", machineID),
			zap.String("boot", "PXE"),
			zap.Error(err),
		)
	}
}
