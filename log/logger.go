package log

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"
)

var log *zap.Logger

func InitDefault() {
	os.Setenv(zapup.KeyFieldApp, "Metal-Core")
	log = zapup.MustRootLogger()
}

func InitConsoleEncoder(cfg *domain.Config) {
	logLevel := strings.ToLower(cfg.LogLevel)

	level := zap.NewAtomicLevel()
	level.UnmarshalText([]byte(logLevel))

	log = Get().WithOptions(
		zap.WrapCore(
			func(zapcore.Core) zapcore.Core {
				return zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), level)
			}))
}

func Get() *zap.Logger {
	if log == nil {
		InitDefault()
	}
	return log
}
