package main

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/version"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"os"
	"strings"
)

var (
	cfg      domain.Config
	zlog     = zapup.MustRootLogger()
	sugarlog = zlog.Sugar()
)

func main() {
	srv := core.NewService(&cfg)
	bus.NewConsumer(zlog, cfg.MQAddress).
		MustRegister("device", "rack1").
		Consume(domain.DeviceEvent{}, func(message interface{}) error {
			evt := message.(*domain.DeviceEvent)
			sugarlog.Info("got message", "event", *evt)
			return nil
		}, 5)
	srv.RunServer()
}

func init() {
	os.Setenv(zapup.KeyLogLevel, strings.ToLower(cfg.LogLevel))
	zapup.MustRootLogger().Info("Metal-Core version",
		zap.Any("version", version.V),
	)
	if err := envconfig.Process("METAL_CORE", &cfg); err != nil {
		zapup.MustRootLogger().Fatal("Bad configuration",
			zap.Error(err),
		)
		os.Exit(1)
	}
	cfg.Log()
}
