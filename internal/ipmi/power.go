package ipmi

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

// PowerOnMachine sets the power of the machine to ON
func PowerOnMachine(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine Power ON",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	err = client.Control(goipmi.ControlPowerUp)
	if err != nil {
		return err
	}

	return nil
}

// PowerOffMachine sets the power of the machine to OFF
func PowerOffMachine(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine Power OFF",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	err = client.Control(goipmi.ControlPowerDown)
	if err != nil {
		return err
	}

	return nil
}

// PowerResetMachine resets the power of the machine
func PowerResetMachine(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine Power RESET",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	err = client.Control(goipmi.ControlPowerHardReset)
	if err != nil {
		return err
	}

	return nil
}

// PowerCycleMachine cycles the power of the machine
func PowerCycleMachine(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine Power CYCLE",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	err = client.Control(goipmi.ControlPowerCycle)
	if err != nil {
		return err
	}

	return nil
}

// PowerOnChassisIdentifyLED powers the machine chassis identify LED on indefinitely (raw 0x00 0x04 0x00 0x01)
func PowerOnChassisIdentifyLED(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine chassis identify LED Power ON",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	err = sendChassisIdentifyRaw(client, 0x00, 0x01)
	if err != nil {
		return err
	}

	return nil
}

// PowerOffChassisIdentifyLED powers the machine chassis identify LED off (raw 0x00 0x04 0x00)
func PowerOffChassisIdentifyLED(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Machine chassis identify LED Power OFF",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	err = sendChassisIdentifyRaw(client, 0x00, 0x00)
	if err != nil {
		return err
	}

	return nil
}
