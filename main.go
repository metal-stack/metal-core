package main

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/core"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

var (
	lvls map[string]log.Level
	cfg  domain.Config
	srv  core.Service
)

func main() {
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

	if err := envconfig.Process("metal_core", &cfg); err != nil {
		log.Error("Configuration error", "error", err)
		os.Exit(1)
	}
	log.SetLevel(fetchLogLevel(cfg.LogLevel))
	cfg.Log()

	srv = core.NewService(cfg)
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
