package ipmi

import (
	"github.com/metal-stack/go-hal"
	"github.com/metal-stack/go-hal/connect"
	halzap "github.com/metal-stack/go-hal/pkg/logger/zap"
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
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(zapup.MustRootLogger().Sugar()))
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
