package ipmi

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

func PowerOn(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Powering up",
		zap.String("hostname", cfg.Hostname),
	)

	err = client.Control(goipmi.ControlPowerUp)
	if err != nil {
		return err
	}
	return nil
}

func PowerOff(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Powering off",
		zap.String("hostname", cfg.Hostname),
	)

	err = client.Control(goipmi.ControlPowerDown)
	if err != nil {
		return err
	}
	return nil
}
