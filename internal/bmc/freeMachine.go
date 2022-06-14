package bmc

import (
	"github.com/metal-stack/go-hal"
)

func (h *BMCService) FreeMachine(event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("reinstall", "error", err)
		return
	}
	err = outBand.BootFrom(hal.BootTargetPXE)
	if err != nil {
		h.log.Sugar().Errorw("reinstall", "error", err)
		return
	}

	err = outBand.PowerCycle()
	if err != nil {
		h.log.Sugar().Errorw("reinstall", "error", err)
		return
	}
}
