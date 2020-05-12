package ipmi

import (
	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/go-hal/detect"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func SetBootPXE(cfg *domain.IPMIConfig) error {
	return boot(cfg, hal.BootTargetPXE)
}

func SetBootDisk(cfg *domain.IPMIConfig) error {
	return boot(cfg, hal.BootTargetDisk)
}

func SetBootBios(cfg *domain.IPMIConfig) error {
	return boot(cfg, hal.BootTargetBIOS)
}

func boot(cfg *domain.IPMIConfig, target hal.BootTarget) error {
	outBand, err := detect.ConnectOutBand(cfg.Hostname, cfg.User(), cfg.Password(), cfg.Compliance)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to outband connect",
			zap.String("hostname", cfg.Hostname),
			zap.String("MAC", cfg.Mac()),
			zap.Error(err),
		)
		return err
	}

	zapup.MustRootLogger().Info("Setting boot machine to boot from",
		zap.String("device", target.String()),
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.BootFrom(target)
}
