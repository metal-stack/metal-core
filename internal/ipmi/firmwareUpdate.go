package ipmi

import (
	"github.com/metal-stack/go-hal/connect"
	"github.com/metal-stack/go-hal/pkg/api"
	halzap "github.com/metal-stack/go-hal/pkg/logger/zap"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
	"go.uber.org/zap"
)

func UpdateBios(cfg *domain.IPMIConfig, revision string, s3Cfg *api.S3Config) error {
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

	partNumber := ""
	if cfg.Ipmi != nil && cfg.Ipmi.Fru != nil {
		partNumber = cfg.Ipmi.Fru.BoardPartNumber
	}

	zapup.MustRootLogger().Info("Updating machine BIOS",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
		zap.String("BoardPartNumber", partNumber),
		zap.String("Revision", revision),
	)

	return outBand.UpdateBIOS(partNumber, revision, s3Cfg)
}

func UpdateBmc(cfg *domain.IPMIConfig, revision string, s3Cfg *api.S3Config) error {
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

	partNumber := ""
	if cfg.Ipmi != nil && cfg.Ipmi.Fru != nil {
		partNumber = cfg.Ipmi.Fru.BoardPartNumber
	}

	zapup.MustRootLogger().Info("Updating machine bmc",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
		zap.String("BoardPartNumber", partNumber),
		zap.String("Revision", revision),
	)

	return outBand.UpdateBMC(partNumber, revision, s3Cfg)
}
