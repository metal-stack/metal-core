package ipmi

import (
	"github.com/metal-stack/go-hal/connect"
	"github.com/metal-stack/go-hal/pkg/api"
	halzap "github.com/metal-stack/go-hal/pkg/logger/zap"
	"github.com/metal-stack/metal-core/pkg/domain"
	"go.uber.org/zap"
)

func UpdateBios(log *zap.Logger, cfg *domain.IPMIConfig, revision string, s3Cfg *api.S3Config) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		log.Error("unable to outband connect",
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

	log.Info("updating machine BIOS",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
		zap.String("BoardPartNumber", partNumber),
		zap.String("Revision", revision),
	)

	return outBand.UpdateBIOS(partNumber, revision, s3Cfg)
}

func UpdateBmc(log *zap.Logger, cfg *domain.IPMIConfig, revision string, s3Cfg *api.S3Config) error {
	host, port, user, password := cfg.IPMIConnection()
	outBand, err := connect.OutBand(host, port, user, password, halzap.New(log.Sugar()))
	if err != nil {
		log.Error("unable to outband connect",
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

	log.Info("updating machine bmc",
		zap.String("hostname", cfg.Hostname),
		zap.String("MAC", cfg.Mac()),
		zap.String("BoardPartNumber", partNumber),
		zap.String("Revision", revision),
	)

	return outBand.UpdateBMC(partNumber, revision, s3Cfg)
}
