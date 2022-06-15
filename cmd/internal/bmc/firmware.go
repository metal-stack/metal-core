package bmc

import (
	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/go-hal/pkg/api"
)

func (h *BMCService) UpdateBios(revision, description string, s3Cfg *api.S3Config, fru Fru, outBand hal.OutBand) {
	err := outBand.UpdateBIOS(fru.BoardPartNumber, revision, s3Cfg)
	if err != nil {
		h.log.Errorw("updatebios", "error", err)
		return
	}
}

func (h *BMCService) UpdateBmc(revision, description string, s3Cfg *api.S3Config, fru Fru, outBand hal.OutBand) {
	err := outBand.UpdateBMC(fru.BoardPartNumber, revision, s3Cfg)
	if err != nil {
		h.log.Errorw("updatebmc", "error", err)
		return
	}
}
