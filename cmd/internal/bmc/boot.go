package bmc

import (
	"github.com/metal-stack/go-hal"
)

func (h *BMCService) PowerBootBiosMachine(outBand hal.OutBand) {
	err := outBand.BootFrom(hal.BootTargetBIOS)
	if err != nil {
		h.log.Errorw("power boot bios", "error", err)
		return
	}
}

func (h *BMCService) PowerBootDiskMachine(outBand hal.OutBand) {
	err := outBand.BootFrom(hal.BootTargetDisk)
	if err != nil {
		h.log.Errorw("power boot disk", "error", err)
		return
	}
}

func (h *BMCService) PowerBootPxeMachine(outBand hal.OutBand) {
	err := outBand.BootFrom(hal.BootTargetPXE)
	if err != nil {
		h.log.Errorw("power boot pxe", "error", err)
		return
	}
}
