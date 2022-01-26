package ipmi

import (
	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/go-hal/connect"
	halzap "github.com/metal-stack/go-hal/pkg/logger/zap"
	"github.com/metal-stack/metal-core/pkg/domain"
	"go.uber.org/zap"
)

func SetBootPXE(log *zap.Logger, cfg *domain.IPMIConfig) error {
	return boot(log, cfg, hal.BootTargetPXE)
}

func SetBootDisk(log *zap.Logger, cfg *domain.IPMIConfig) error {
	return boot(log, cfg, hal.BootTargetDisk)
}

func SetBootBios(log *zap.Logger, cfg *domain.IPMIConfig) error {
	return boot(log, cfg, hal.BootTargetBIOS)
}

func boot(log *zap.Logger, cfg *domain.IPMIConfig, target hal.BootTarget) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		log.Error("Unable to outband connect",
			zap.String("hostname", cfg.Hostname),
			zap.String("MAC", cfg.Mac()),
			zap.Error(err),
		)
		return err
	}

	log.Info("Setting boot machine to boot from",
		zap.String("device", target.String()),
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
	)

	return outBand.BootFrom(target)
}
