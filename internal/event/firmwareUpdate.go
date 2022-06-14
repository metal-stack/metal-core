package event

import (
	"github.com/metal-stack/go-hal/pkg/api"
	"github.com/metal-stack/metal-core/pkg/domain"
)

func (h *eventHandler) UpdateBios(revision, description string, s3Cfg *api.S3Config, event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("updatebios", "error", err)
		return
	}
	if event.IPMI == nil {
		h.Log.Sugar().Errorw("updatebios ipmi config is nil")
		return
	}

	err = outBand.UpdateBIOS(event.IPMI.Fru.BoardPartNumber, revision, s3Cfg)
	if err != nil {
		h.Log.Sugar().Errorw("updatebios", "error", err)
		return
	}
}

func (h *eventHandler) UpdateBmc(revision, description string, s3Cfg *api.S3Config, event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("updatebios", "error", err)
		return
	}
	if event.IPMI == nil {
		h.Log.Sugar().Errorw("updatebios ipmi config is nil")
		return
	}

	err = outBand.UpdateBMC(event.IPMI.Fru.BoardPartNumber, revision, s3Cfg)
	if err != nil {
		h.Log.Sugar().Errorw("updatebios", "error", err)
		return
	}
}
