package event

func (h *EventHandler) PowerOnMachine(event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("power on", "error", err)
		return
	}
	err = outBand.PowerOn()
	if err != nil {
		h.log.Sugar().Errorw("power on", "error", err)
		return
	}
}

func (h *EventHandler) PowerOffMachine(event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("power off", "error", err)
		return
	}
	err = outBand.PowerOff()
	if err != nil {
		h.log.Sugar().Errorw("power off", "error", err)
		return
	}
}

func (h *EventHandler) PowerResetMachine(event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("power reset", "error", err)
		return
	}
	err = outBand.PowerReset()
	if err != nil {
		h.log.Sugar().Errorw("power reset", "error", err)
		return
	}
}

func (h *EventHandler) PowerCycleMachine(event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("power cycle", "error", err)
		return
	}
	err = outBand.PowerCycle()
	if err != nil {
		h.log.Sugar().Errorw("power cycle", "error", err)
		return
	}
}

func (h *EventHandler) PowerOnChassisIdentifyLED(event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("power led on", "error", err)
		return
	}
	err = outBand.IdentifyLEDOn()
	if err != nil {
		h.log.Sugar().Errorw("power led on", "error", err)
		return
	}
}

func (h *EventHandler) PowerOffChassisIdentifyLED(event MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.log.Sugar())
	if err != nil {
		h.log.Sugar().Errorw("power led off", "error", err)
		return
	}
	err = outBand.IdentifyLEDOff()
	if err != nil {
		h.log.Sugar().Errorw("power led off", "error", err)
		return
	}
}
