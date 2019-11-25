package ipmi

import (
	"fmt"
	"strings"

	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

const (
	LegacyQualifier = uint8(0xff)

	PXEQualifier       = uint8(0x04)
	DefaultHDQualifier = uint8(0x08)
	BIOSQualifier      = uint8(0x18)

	UEFIQualifier = uint8(0xe0)

	UEFIPXEQualifier = uint8(0x04)
	UEFIHDQualifier  = uint8(0x24)
)

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

	manufacturer := strings.ToLower(strings.TrimSpace(cfg.Ipmi.Fru.ProductManufacturer))
	if strings.Contains(manufacturer, "supermicro") {
		return viaIPMIRaw
	}

	return viaIPMI
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
// https://git.f-i-ts.de/cloud-native/metal/smcipmitool/blob/master/com/supermicro/ipmi/IPMIChassisCommand.java#L265
func viaIPMIRaw(cfg *domain.IPMIConfig, dev goipmi.BootDevice) error {
	client, err := openClientConnection(cfg)
	if err != nil {
		return err
	}

	/*
	   Set boot order to UEFI PXE persistently:
	   raw 0x00 0x08 0x05 0xe0 0x04 0x00 0x00 0x00  (conforms to IPMI 2.0 as well as SMCIPMITool)

	   Set boot order to UEFI HD persistently:
	   raw 0x00 0x08 0x05 0xe0 0x08 0x00 0x00 0x00  (IPMI 2.0)
	   raw 0x00 0x08 0x05 0xe0 0x24 0x00 0x00 0x00  (SMCIPMITool)

	   Set boot order to (UEFI) BIOS persistently:
	   raw 0x00 0x08 0x05 0xe0 0x18 0x00 0x00 0x00  (IPMI 2.0   , UEFI BIOS)
	   raw 0x00 0x08 0x05 0xff 0x18 0x00 0x00 0x00  (SMCIPMITool, legacy BIOS)

	   See https://git.f-i-ts.de/cloud-native/metal/metal/issues/73#note_151375
	*/

	var uefiQualifier, bootDevQualifier uint8
	// Conforms to SMCIPMITool
	switch dev {
	case goipmi.BootDevicePxe:
		uefiQualifier = UEFIQualifier
		bootDevQualifier = UEFIPXEQualifier
	case goipmi.BootDeviceDisk:
		uefiQualifier = UEFIQualifier
		bootDevQualifier = UEFIHDQualifier // use DefaultHDQualifier for IPMI 2.0 compatibility
	case goipmi.BootDeviceBios:
		uefiQualifier = LegacyQualifier // use UEFIQualifier for IPMI 2.0 compatibility
		bootDevQualifier = BIOSQualifier
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

	err = sendSystemBootRaw(client, goipmi.BootParamBootFlags, uefiQualifier, bootDevQualifier, 0x00, 0x00, 0x00)
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
