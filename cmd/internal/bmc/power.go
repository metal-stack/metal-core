package bmc

func (h *BMCService) PowerOnMachine(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power on", "error", err)
		return
	}
	err = outBand.PowerOn()
	if err != nil {
		h.log.Errorw("power on", "error", err)
		return
	}
}

func (h *BMCService) PowerOffMachine(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power off", "error", err)
		return
	}
	err = outBand.PowerOff()
	if err != nil {
		h.log.Errorw("power off", "error", err)
		return
	}
}

func (h *BMCService) PowerResetMachine(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power reset", "error", err)
		return
	}
	err = outBand.PowerReset()
	if err != nil {
		h.log.Errorw("power reset", "error", err)
		return
	}
}

func (h *BMCService) PowerCycleMachine(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power cycle", "error", err)
		return
	}
	err = outBand.PowerCycle()
	if err != nil {
		h.log.Errorw("power cycle", "error", err)
		return
	}
}

func (h *BMCService) PowerOnChassisIdentifyLED(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power led on", "error", err)
		return
	}
	err = outBand.IdentifyLEDOn()
	if err != nil {
		h.log.Errorw("power led on", "error", err)
		return
	}
}

func (h *BMCService) PowerOffChassisIdentifyLED(event *MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log)
	if err != nil {
		h.log.Errorw("power led off", "error", err)
		return
	}
	err = outBand.IdentifyLEDOff()
	if err != nil {
		h.log.Errorw("power led off", "error", err)
		return
	}
}
