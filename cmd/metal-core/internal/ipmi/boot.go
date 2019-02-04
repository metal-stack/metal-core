package ipmi

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

func SetBootDevPxe(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Setting boot device to PXE boot",
		zap.String("hostname", cfg.Hostname),
	)

	err = client.SetBootDevice(goipmi.BootDevicePxe)
	if err != nil {
		return err
	}
	return nil
}

func SetBootDevHd(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Setting boot device to HD boot",
		zap.String("hostname", cfg.Hostname),
	)

	err = client.SetBootDevice(goipmi.BootDeviceDisk)
	if err != nil {
		return err
	}
	return nil
}
