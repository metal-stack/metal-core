package ipmi

import (
	"github.com/metal-stack/go-hal/connect"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

// PowerOnMachine sets the power of the machine to ON
func PowerOnMachine(cfg *domain.IPMIConfig) error {
	outBand, err := connect.OutBand(cfg.IPMIConnection())
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine Power ON",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.PowerOn()
}

// PowerOffMachine sets the power of the machine to OFF
func PowerOffMachine(cfg *domain.IPMIConfig) error {
	outBand, err := connect.OutBand(cfg.IPMIConnection())
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine Power OFF",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.PowerOff()
}

// PowerResetMachine resets the power of the machine
func PowerResetMachine(cfg *domain.IPMIConfig) error {
	outBand, err := connect.OutBand(cfg.IPMIConnection())
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine Power RESET",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)
	return outBand.PowerReset()
}

// PowerCycleMachine cycles the power of the machine
func PowerCycleMachine(cfg *domain.IPMIConfig) error {
	outBand, err := connect.OutBand(cfg.IPMIConnection())
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine Power CYCLE",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.PowerCycle()
}

// PowerOnChassisIdentifyLED powers the machine chassis identify LED on indefinitely (raw 0x00 0x04 0x00 0x01)
func PowerOnChassisIdentifyLED(cfg *domain.IPMIConfig) error {
	outBand, err := connect.OutBand(cfg.IPMIConnection())
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine chassis identify LED Power ON",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.IdentifyLEDOn()
}

// PowerOffChassisIdentifyLED powers the machine chassis identify LED off (raw 0x00 0x04 0x00)
func PowerOffChassisIdentifyLED(cfg *domain.IPMIConfig) error {
	outBand, err := connect.OutBand(cfg.IPMIConnection())
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine chassis identify LED Power OFF",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.IdentifyLEDOff()
}
