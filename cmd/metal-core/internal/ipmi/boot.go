package ipmi

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

func SetBootMachinePXE(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Setting boot machine to PXE boot",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return client.SetBootDevice(goipmi.BootDevicePxe)
}

func SetBootMachineHD(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Setting boot machine to HD boot",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return client.SetBootDevice(goipmi.BootDeviceDisk)
}

func SetBootMachineBios(cfg *domain.IPMIConfig) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Setting boot machine to BIOS boot",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return client.SetBootDevice(goipmi.BootDeviceBios)
}
