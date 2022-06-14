package event

import (
	"github.com/metal-stack/go-hal/pkg/api"
)

func (h *EventService) UpdateBios(revision, description string, s3Cfg *api.S3Config, event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("updatebios", "error", err)
		return
	}
	if event.IPMI == nil {
		h.log.Sugar().Errorw("updatebios ipmi config is nil")
		return
	}

	err = outBand.UpdateBIOS(event.IPMI.Fru.BoardPartNumber, revision, s3Cfg)
	if err != nil {
		h.log.Sugar().Errorw("updatebios", "error", err)
		return
	}
}

func (h *EventService) UpdateBmc(revision, description string, s3Cfg *api.S3Config, event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("updatebios", "error", err)
		return
	}
	if event.IPMI == nil {
		h.log.Sugar().Errorw("updatebios ipmi config is nil")
		return
	}

	err = outBand.UpdateBMC(event.IPMI.Fru.BoardPartNumber, revision, s3Cfg)
	if err != nil {
		h.log.Sugar().Errorw("updatebios", "error", err)
		return
	}
}
