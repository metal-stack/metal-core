package bmc

import (
	"github.com/metal-stack/go-hal"
)

func (h *BMCService) PowerBootBiosMachine(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power boot bios", "error", err)
		return
	}
	err = outBand.BootFrom(hal.BootTargetBIOS)
	if err != nil {
		h.log.Errorw("power boot bios", "error", err)
		return
	}
}

func (h *BMCService) PowerBootDiskMachine(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power boot disk", "error", err)
		return
	}
	err = outBand.BootFrom(hal.BootTargetDisk)
	if err != nil {
		h.log.Errorw("power boot disk", "error", err)
		return
	}
}

func (h *BMCService) PowerBootPxeMachine(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power boot pxe", "error", err)
		return
	}
	err = outBand.BootFrom(hal.BootTargetPXE)
	if err != nil {
		h.log.Errorw("power boot pxe", "error", err)
		return
	}
}
