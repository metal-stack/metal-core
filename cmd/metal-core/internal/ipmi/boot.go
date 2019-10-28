package ipmi

import (
	"fmt"

	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

const (
	BootDevPersistentEfiQualifier = uint8(0xe0)
	HardDiskQualifier             = uint8(0x08)
	BiosQualifier                 = uint8(0x24)
	PXEQualifier                  = uint8(0x04)
)

var diskQualifiers = map[string]uint8{
	"SYS-2029BT-HNTR":     HardDiskQualifier,
	"SYS-2029BT-HNR":      BiosQualifier,
	"SSG-5049P-E1CR45H":   HardDiskQualifier,
	"MBI-6418A-T5H":       HardDiskQualifier,
	"MBI-6219G-T7LX-PACK": HardDiskQualifier,
	"vagrant":             HardDiskQualifier,
}

func diskQualifier(cfg *domain.IPMIConfig) uint8 {
	if cfg.Ipmi.Fru == nil {
		return HardDiskQualifier
	}

	diskQualifier, ok := diskQualifiers[cfg.Ipmi.Fru.ProductPartNumber]
	if !ok {
		return HardDiskQualifier
	}

	return diskQualifier
}

var bootMethods = map[string]func(*domain.IPMIConfig, goipmi.BootDevice) error{
	"SYS-2029BT-HNTR":     viaIPMIRaw,
	"SYS-2029BT-HNR":      viaIPMIRaw,
	"SSG-5049P-E1CR45H":   viaIPMIRaw,
	"MBI-6418A-T5H":       viaIPMIRaw,
	"MBI-6219G-T7LX-PACK": viaIPMIRaw,
	"vagrant":             viaIPMI,
}

func SetBootPXE(cfg *domain.IPMIConfig) error {
	return boot(cfg)(cfg, goipmi.BootDevicePxe)
}

func SetBootDisk(cfg *domain.IPMIConfig) error {
	return boot(cfg)(cfg, goipmi.BootDeviceDisk)
}

func SetBootBios(cfg *domain.IPMIConfig) error {
	return boot(cfg)(cfg, goipmi.BootDeviceBios)
}

func boot(cfg *domain.IPMIConfig) func(cfg *domain.IPMIConfig, dev goipmi.BootDevice) error {
	if cfg.Ipmi.Fru == nil {
		return viaIPMI
	}

	bootMethod, ok := bootMethods[cfg.Ipmi.Fru.ProductPartNumber]
	if !ok {
		return viaIPMI
	}

	return bootMethod
}

func viaIPMI(cfg *domain.IPMIConfig, dev goipmi.BootDevice) error {
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

// viaIPMIRaw is a modified wrapper around
// goipmi.SetSystemBootOptionsRequest to configure the BootDevice per section 28.12:
// https://www.intel.com/content/dam/www/public/us/en/documents/product-briefs/ipmi-second-gen-interface-spec-v2-rev1-1.pdf
// We send modified raw parameters according to:
// https://www.supermicro.com/support/faqs/faq.cfm?faq=25559
func viaIPMIRaw(cfg *domain.IPMIConfig, dev goipmi.BootDevice) error {
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

	var bootDevQualifier uint8
	switch dev {
	case goipmi.BootDeviceDisk:
		bootDevQualifier = diskQualifier(cfg)
	case goipmi.BootDeviceBios:
		bootDevQualifier = BiosQualifier
	case goipmi.BootDevicePxe:
		bootDevQualifier = PXEQualifier
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
	err = sendSystemBootRaw(client, goipmi.BootParamBootFlags, BootDevPersistentEfiQualifier, bootDevQualifier, 0x00, 0x00, 0x00)
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
