package main

import (
	"os"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/version"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

var srv core.Service

func main() {
	srv.RunServer()
}

func init() {
	cfg := &domain.Config{}
	if err := envconfig.Process("METAL_CORE", cfg); err != nil {
		zapup.MustRootLogger().Fatal("Bad configuration",
			zap.Error(err),
		)
		os.Exit(1)
	}

	os.Setenv(zapup.KeyFieldApp, "Metal-Core")
	if cfg.ConsoleLogging {
		os.Setenv(zapup.KeyLogEncoding, "console")
	}

	initService(cfg)
	initConsumer(cfg)

	zapup.MustRootLogger().Info("Metal-Core Version",
		zap.Any("version", version.V),
	)
	zapup.MustRootLogger().Info("Configuration",
		zap.String("LogLevel", cfg.LogLevel),
		zap.String("BindAddress", cfg.BindAddress),
		zap.String("IP", cfg.IP),
		zap.Int("Port", cfg.Port),
		zap.String("API-Protocol", cfg.ApiProtocol),
		zap.String("API-IP", cfg.ApiIP),
		zap.Int("API-Port", cfg.ApiPort),
		zap.String("HammerImagePrefix", cfg.HammerImagePrefix),
	)
}

func initService(cfg *domain.Config) {
	srv = core.NewService(cfg)
}

func initConsumer(cfg *domain.Config) {
	bus.NewConsumer(zapup.MustRootLogger(), cfg.MQAddress).
		MustRegister("device", "rack1").
		Consume(domain.DeviceEvent{}, func(message interface{}) error {
			evt := message.(*domain.DeviceEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.Any("event", evt),
			)
			if evt.Type == domain.DELETE {
				srv.FreeDevice(evt.Old)
			}
			return nil
		}, 5)
}
