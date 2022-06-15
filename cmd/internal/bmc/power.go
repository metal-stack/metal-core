package bmc

import "github.com/metal-stack/go-hal"

func (h *BMCService) PowerOnMachine(outBand hal.OutBand) {
	err := outBand.PowerOn()
	if err != nil {
		h.log.Errorw("power on", "error", err)
		return
	}
}

func (h *BMCService) PowerOffMachine(outBand hal.OutBand) {
	err := outBand.PowerOff()
	if err != nil {
		h.log.Errorw("power off", "error", err)
		return
	}
}

func (h *BMCService) PowerResetMachine(outBand hal.OutBand) {
	err := outBand.PowerReset()
	if err != nil {
		h.log.Errorw("power reset", "error", err)
		return
	}
}

func (h *BMCService) PowerCycleMachine(outBand hal.OutBand) {
	err := outBand.PowerCycle()
	if err != nil {
		h.log.Errorw("power cycle", "error", err)
		return
	}
}

func (h *BMCService) PowerOnChassisIdentifyLED(outBand hal.OutBand) {
	err := outBand.IdentifyLEDOn()
	if err != nil {
		h.log.Errorw("power led on", "error", err)
		return
	}
}

func (h *BMCService) PowerOffChassisIdentifyLED(outBand hal.OutBand) {
	err := outBand.IdentifyLEDOff()
	if err != nil {
		h.log.Errorw("power led off", "error", err)
		return
	}
}
