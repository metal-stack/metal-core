package main

import (
	"os"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

var (
	lvls     map[string]log.Level
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
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	lvls = make(map[string]log.Level, 6)
	lvls["DEBUG"] = log.DebugLevel
	lvls["INFO"] = log.InfoLevel
	lvls["WARN"] = log.WarnLevel
	lvls["ERROR"] = log.ErrorLevel
	lvls["FATAL"] = log.FatalLevel
	lvls["PANIC"] = log.PanicLevel

	if err := envconfig.Process("METAL_CORE", &cfg); err != nil {
		log.Fatal("Bad configuration", "error", err)
		os.Exit(1)
	}
	log.SetLevel(fetchLogLevel(cfg.LogLevel))
	cfg.Log()
}

func fetchLogLevel(lvl string) log.Level {
	lvl = strings.ToUpper(lvl)
	for k, v := range lvls {
		if k == lvl {
			return v
		}
	}
	return log.InfoLevel
}
