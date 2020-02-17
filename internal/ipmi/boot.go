package ipmi

import (
	"fmt"

	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
	goipmi "github.com/vmware/goipmi"
	"go.uber.org/zap"
)

const (
	LegacyQualifier = uint8(0xff)

	PXEQualifier       = uint8(0x04)
	DefaultHDQualifier = uint8(0x08) // IPMI 2.0 compatible
	BIOSQualifier      = uint8(0x18)

	UEFIQualifier = uint8(0xe0)

	UEFIPXEQualifier = uint8(0x04)
	UEFIHDQualifier  = uint8(0x24) // SMCIPMITool compatible
)

func SetBootPXE(cfg *domain.IPMIConfig) error {
	return boot(cfg, goipmi.BootDevicePxe, false)
}

func SetBootDisk(cfg *domain.IPMIConfig, devMode bool) error {
	return boot(cfg, goipmi.BootDeviceDisk, devMode)
}

func SetBootBios(cfg *domain.IPMIConfig, devMode bool) error {
	return boot(cfg, goipmi.BootDeviceBios, devMode)
}

// boot is a modified wrapper around
// goipmi.SetSystemBootOptionsRequest to configure the BootDevice per section 28.12:
// https://www.intel.com/content/dam/www/public/us/en/documents/product-briefs/ipmi-second-gen-interface-spec-v2-rev1-1.pdf
// We send modified raw parameters according to:
// https://git.f-i-ts.de/cloud-native/metal/smcipmitool/blob/master/com/supermicro/ipmi/IPMIChassisCommand.java#L265
func boot(cfg *domain.IPMIConfig, dev goipmi.BootDevice, devMode bool) error {
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
	switch dev {
	case goipmi.BootDevicePxe:
		uefiQualifier = UEFIQualifier
		bootDevQualifier = UEFIPXEQualifier
	case goipmi.BootDeviceDisk:
		uefiQualifier = UEFIQualifier
		if devMode {
			bootDevQualifier = DefaultHDQualifier // Conforms to IPMI 2.0
		} else {
			bootDevQualifier = UEFIHDQualifier // Conforms to SMCIPMITool
		}
	case goipmi.BootDeviceBios:
		if devMode {
			uefiQualifier = UEFIQualifier // Conforms to IPMI 2.0
		} else {
			uefiQualifier = LegacyQualifier // Conforms to SMCIPMITool
		}
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
