package main

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/event"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/server"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"github.com/go-openapi/strfmt"
	"os"

	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/version"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/go-openapi/runtime/client"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

var (
	appContext *domain.AppContext
)

func main() {
	appContext.Server().Run()
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

	transport := client.New(fmt.Sprintf("%v:%d", cfg.ApiIP, cfg.ApiPort), "", nil)

	appContext = &domain.AppContext{
		Config:              cfg,
		ApiClientHandler:    api.Handler,
		ServerHandler:       server.Handler,
		EndpointHandler:     endpoint.Handler,
		EventHandlerHandler: event.Handler,
		DeviceClient:        device.New(transport, strfmt.Default),
	}

	initConsumer()
}

func initConsumer() {
	bus.NewConsumer(zapup.MustRootLogger(), appContext.Config.MQAddress).
		MustRegister("device", "rack1").
		Consume(domain.DeviceEvent{}, func(message interface{}) error {
			evt := message.(*domain.DeviceEvent)
			zapup.MustRootLogger().Info("Got message",
				zap.Any("event", evt),
			)
			if evt.Type == domain.DELETE {
				appContext.EventHandler().FreeDevice(evt.Old)
			}
			return nil
		}, 5)
}
