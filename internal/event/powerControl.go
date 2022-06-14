package event

import (
	"github.com/metal-stack/metal-core/pkg/domain"
)

func (h *eventHandler) PowerOnMachine(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power on", "error", err)
		return
	}
	err = outBand.PowerOn()
	if err != nil {
		h.Log.Sugar().Errorw("power on", "error", err)
		return
	}
}

func (h *eventHandler) PowerOffMachine(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power off", "error", err)
		return
	}
	err = outBand.PowerOff()
	if err != nil {
		h.Log.Sugar().Errorw("power off", "error", err)
		return
	}
}

func (h *eventHandler) PowerResetMachine(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power reset", "error", err)
		return
	}
	err = outBand.PowerReset()
	if err != nil {
		h.Log.Sugar().Errorw("power reset", "error", err)
		return
	}
}

func (h *eventHandler) PowerCycleMachine(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power cycle", "error", err)
		return
	}
	err = outBand.PowerCycle()
	if err != nil {
		h.Log.Sugar().Errorw("power cycle", "error", err)
		return
	}
}

func (h *eventHandler) PowerOnChassisIdentifyLED(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power led on", "error", err)
		return
	}
	err = outBand.IdentifyLEDOn()
	if err != nil {
		h.Log.Sugar().Errorw("power led on", "error", err)
		return
	}
}

func (h *eventHandler) PowerOffChassisIdentifyLED(event domain.MachineEvent) {
	outBand, err := outBand(*event.IPMI, h.Log.Sugar())
	if err != nil {
		h.Log.Sugar().Errorw("power led off", "error", err)
		return
	}
	err = outBand.IdentifyLEDOff()
	if err != nil {
		h.Log.Sugar().Errorw("power led off", "error", err)
		return
	}
}
