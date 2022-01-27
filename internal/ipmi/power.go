package ipmi

import (
	"github.com/metal-stack/go-hal/connect"
	halzap "github.com/metal-stack/go-hal/pkg/logger/zap"
	"github.com/metal-stack/metal-core/pkg/domain"
	"go.uber.org/zap"
)

// PowerOnMachine sets the power of the machine to ON
func PowerOnMachine(log *zap.Logger, cfg *domain.IPMIConfig) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		return err
	}

	log.Info("machine power on",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.PowerOn()
}

// PowerOffMachine sets the power of the machine to OFF
func PowerOffMachine(log *zap.Logger, cfg *domain.IPMIConfig) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		return err
	}

	log.Info("machine power off",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.PowerOff()
}

// PowerResetMachine resets the power of the machine
func PowerResetMachine(log *zap.Logger, cfg *domain.IPMIConfig) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		return err
	}

	log.Info("machine power reset",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)
	return outBand.PowerReset()
}

// PowerCycleMachine cycles the power of the machine
func PowerCycleMachine(log *zap.Logger, cfg *domain.IPMIConfig) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		return err
	}

	log.Info("machine power cycle",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.PowerCycle()
}

// PowerOnChassisIdentifyLED powers the machine chassis identify LED on indefinitely (raw 0x00 0x04 0x00 0x01)
func PowerOnChassisIdentifyLED(log *zap.Logger, cfg *domain.IPMIConfig) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		return err
	}

	log.Info("machine chassis identify led power on",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.IdentifyLEDOn()
}

// PowerOffChassisIdentifyLED powers the machine chassis identify LED off (raw 0x00 0x04 0x00)
func PowerOffChassisIdentifyLED(log *zap.Logger, cfg *domain.IPMIConfig) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		return err
	}

	log.Info("machine chassis identify led power off",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.IdentifyLEDOff()
}
