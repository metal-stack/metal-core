package event

import (
	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/metal-core/pkg/domain"
)

func (h *eventHandler) FreeMachine(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("reinstall", "error", err)
		return
	}
	err = outBand.BootFrom(hal.BootTargetPXE)
	if err != nil {
		h.Log.Sugar().Errorw("reinstall", "error", err)
		return
	}

	err = outBand.PowerCycle()
	if err != nil {
		h.Log.Sugar().Errorw("reinstall", "error", err)
		return
	}
}
