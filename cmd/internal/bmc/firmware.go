package bmc

import (
	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/go-hal/pkg/api"
)

func (h *BMCService) UpdateBios(revision, description string, s3Cfg *api.S3Config, event *MachineEvent, outBand hal.OutBand) {
	err := outBand.UpdateBIOS(event.IPMI.Fru.BoardPartNumber, revision, s3Cfg)
	if err != nil {
		h.log.Errorw("updatebios", "error", err)
		return
	}
}

func (h *BMCService) UpdateBmc(revision, description string, s3Cfg *api.S3Config, event *MachineEvent, outBand hal.OutBand) {
	err := outBand.UpdateBMC(event.IPMI.Fru.BoardPartNumber, revision, s3Cfg)
	if err != nil {
		h.log.Errorw("updatebmc", "error", err)
		return
	}
}
