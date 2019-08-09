package ipmi

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

const (
	BootDevPersistentEfiQualifier = 0xe0
	HardDiskQualifier             = 0x08
	BiosQualifier                 = 0x24
	PxeQualifier                  = 0x04
)

var bootFuncMap = map[string]func(*domain.IPMIConfig, goipmi.BootDevice) error{
	"SYS-2029BT-HNTR":     setBootMachineRaw,
	"SYS-2029BT-HNR":      setBootMachineRaw,
	"SSG-5049P-E1CR45H":   setBootMachineRaw,
	"MBI-6418A-T5H":       setBootMachineRaw,
	"MBI-6219G-T7LX-PACK": setBootMachineRaw,
	"vagrant":             setBootMachineIPMI,
}

func SetBootMachinePXE(cfg *domain.IPMIConfig) error {
	return fetchBootFunc(cfg)(cfg, goipmi.BootDevicePxe)
}

func SetBootMachineDisk(cfg *domain.IPMIConfig) error {
	return fetchBootFunc(cfg)(cfg, goipmi.BootDeviceDisk)
}

func SetBootMachineBios(cfg *domain.IPMIConfig) error {
	return fetchBootFunc(cfg)(cfg, goipmi.BootDeviceBios)
}

func fetchBootFunc(cfg *domain.IPMIConfig) func(cfg *domain.IPMIConfig, dev goipmi.BootDevice) error {
	if cfg.Ipmi.Fru == nil {
		return setBootMachineIPMI
	}

	bootFunc, ok := bootFuncMap[cfg.Ipmi.Fru.ProductPartNumber]
	if !ok {
		return setBootMachineIPMI
	}

	return bootFunc
}

func setBootMachineIPMI(cfg *domain.IPMIConfig, dev goipmi.BootDevice) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	zapup.MustRootLogger().Info("Setting boot machine",
		zap.String("device", dev.String()),
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return client.SetBootDevice(dev)
}

// setBootMachineRaw is a modified wrapper around
// goipmi.SetSystemBootOptionsRequest to configure the BootDevice per section 28.12:
// https://www.intel.com/content/dam/www/public/us/en/documents/product-briefs/ipmi-second-gen-interface-spec-v2-rev1-1.pdf
// We send modified raw parameters according to:
// https://www.supermicro.com/support/faqs/faq.cfm?faq=25559
func setBootMachineRaw(cfg *domain.IPMIConfig, dev goipmi.BootDevice) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	/*
		Set 1st boot device to efi hard drive persistently:
		raw 0x00 0x08 0x05 0xe0 0x08 0x00 0x00 0x00

		Set 1st boot device to efi BIOS persistently:
		raw 0x00 0x08 0x05 0xe0 0x24 0x00 0x00 0x00

		Set 1st boot device to efi PXE persistently:
		raw 0x00 0x08 0x05 0xe0 0x04 0x00 0x00 0x00
	*/

	var bootDev uint8
	switch dev {
	case goipmi.BootDeviceDisk:
		bootDev = HardDiskQualifier
	case goipmi.BootDeviceBios:
		bootDev = BiosQualifier
	case goipmi.BootDevicePxe:
		bootDev = PxeQualifier
	default:
		return fmt.Errorf("unsupported boot device:%s", dev.String())
	}

	zapup.MustRootLogger().Info("Setting boot machine to boot from",
		zap.String("device", dev.String()),
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	useProgress := true
	// set set-in-progress flag
	err = sendSystemBootRaw(client, goipmi.BootParamSetInProgress, 0x01)
	if err != nil {
		useProgress = false
	}

	err = sendSystemBootRaw(client, goipmi.BootParamInfoAck, 0x01, 0x01)
	if err != nil {
		if useProgress {
			// set-in-progress = set-complete
			_ = sendSystemBootRaw(client, goipmi.BootParamSetInProgress, 0x00)
		}
		return err
	}

	// 0x00 0x08 0x05 0xe0 [0x08|0x024|0x04] 0x00 0x00 0x00
	err = sendSystemBootRaw(client, goipmi.BootParamBootFlags, BootDevPersistentEfiQualifier, bootDev, 0x00, 0x00, 0x00)
	if err == nil {
		if useProgress {
			// set-in-progress = commit-write
			_ = sendSystemBootRaw(client, goipmi.BootParamSetInProgress, 0x02)
		}
	}

	if useProgress {
		// set-in-progress = set-complete
		_ = sendSystemBootRaw(client, goipmi.BootParamSetInProgress, 0x00)
	}

	return err
}
