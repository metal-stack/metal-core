package bmc

import (
	"github.com/metal-stack/go-hal"
)

func (h *BMCService) FreeMachine(outBand hal.OutBand) {
	err := outBand.BootFrom(hal.BootTargetPXE)
	if err != nil {
		h.log.Errorw("freemachine", "error", err)
		return
	}

	err = outBand.PowerCycle()
	if err != nil {
		h.log.Errorw("freemachine", "error", err)
		return
	}
}
