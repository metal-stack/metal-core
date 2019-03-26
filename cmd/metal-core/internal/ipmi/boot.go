package ipmi

import (
	"fmt"
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

// SetBootDevice is a modified wrapper around
// SetSystemBootOptionsRequest to configure the BootDevice
// per section 28.12 - table 28
// We send modified raw parameters according to:
// https://www.supermicro.com/support/faqs/faq.cfm?faq=25559
func SetBootDevice(cfg *domain.IPMIConfig, dev goipmi.BootDevice) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	/*
		Set 1st boot device to uefi hard drive persistently:
		raw 0x0 0x8 0x05 0xe0 0x24 0x0 0x0 0x0

		Set 1st boot device to uefi pxe persistently:
		raw 0x0 0x8 0x05 0xe0 0x04 0x0 0x0 0x0
	*/

	const SupermicroBootDevQualifier = 0xe0
	const SuperMicroBootDiskQualifier = 0x24
	const SuperMicroBootPxeQualifier = 0x04
	var superMicroBootDevice uint8
	switch dev {
	case goipmi.BootDeviceDisk:
		superMicroBootDevice = SuperMicroBootDiskQualifier
	case goipmi.BootDevicePxe:
		superMicroBootDevice = SuperMicroBootPxeQualifier
	default:
		return fmt.Errorf("unsupported boot device:%v", dev)
	}

	zapup.MustRootLogger().Info("Setting boot machine to boot from",
		zap.String("device", dev.String()),
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	useProgress := true
	// set set-in-progress flag
	err = setBootParam(client, goipmi.BootParamSetInProgress, 0x01)
	if err != nil {
		useProgress = false
	}

	err = setBootParam(client, goipmi.BootParamInfoAck, 0x01, 0x01)
	if err != nil {
		if useProgress {
			// set-in-progress = set-complete
			_ = setBootParam(client, goipmi.BootParamSetInProgress, 0x00)
		}
		return err
	}

	// 0x00 0x08 0x05
	err = setBootParam(client, goipmi.BootParamBootFlags, SupermicroBootDevQualifier, superMicroBootDevice, 0x00, 0x00, 0x00)
	if err == nil {
		if useProgress {
			// set-in-progress = commit-write
			_ = setBootParam(client, goipmi.BootParamSetInProgress, 0x02)
		}
	}

	if useProgress {
		// set-in-progress = set-complete
		_ = setBootParam(client, goipmi.BootParamSetInProgress, 0x00)
	}

	return err
}

func setBootParam(client *goipmi.Client, param uint8, data ...uint8) error {
	r := &goipmi.Request{
		goipmi.NetworkFunctionChassis,      // 0x00
		goipmi.CommandSetSystemBootOptions, // 0x08
		&goipmi.SetSystemBootOptionsRequest{
			Param: param,
			Data:  data,
		},
	}
	return client.Send(r, &goipmi.SetSystemBootOptionsResponse{})
}
