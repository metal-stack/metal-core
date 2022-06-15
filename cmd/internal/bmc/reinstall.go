package bmc

import (
	"github.com/metal-stack/go-hal"
)

func (h *BMCService) ReinstallMachine(outBand hal.OutBand) {
	err := outBand.BootFrom(hal.BootTargetPXE)
	if err != nil {
		h.log.Errorw("reinstall", "error", err)
		return
	}

	err = outBand.PowerReset()
	if err != nil {
		h.log.Errorw("reinstall", "error", err)
		return
	}
}
