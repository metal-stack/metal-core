package event

import (
	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/metal-core/pkg/domain"
)

func (h *eventHandler) PowerBootBiosMachine(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power boot bios", "error", err)
		return
	}
	err = outBand.BootFrom(hal.BootTargetBIOS)
	if err != nil {
		h.Log.Sugar().Errorw("power boot bios", "error", err)
		return
	}
}

func (h *eventHandler) PowerBootDiskMachine(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power boot disk", "error", err)
		return
	}
	err = outBand.BootFrom(hal.BootTargetDisk)
	if err != nil {
		h.Log.Sugar().Errorw("power boot disk", "error", err)
		return
	}
}

func (h *eventHandler) PowerBootPxeMachine(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power boot pxe", "error", err)
		return
	}
	err = outBand.BootFrom(hal.BootTargetPXE)
	if err != nil {
		h.Log.Sugar().Errorw("power boot pxe", "error", err)
		return
	}
}
