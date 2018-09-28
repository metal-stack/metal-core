package main

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/metal-api"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/server"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

var logLevels map[string]log.Level

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	logLevels = make(map[string]log.Level, 6)
	logLevels["DEBUG"] = log.DebugLevel
	logLevels["INFO"] = log.InfoLevel
	logLevels["WARN"] = log.WarnLevel
	logLevels["ERROR"] = log.ErrorLevel
	logLevels["FATAL"] = log.FatalLevel
	logLevels["PANIC"] = log.PanicLevel
}

func main() {
	var config domain.Config
	if err := envconfig.Process("metalcore", &config); err != nil {
		log.Error("Configuration error", "error", err)
		os.Exit(1)
	}
	log.SetLevel(fetchLogLevel(config.LogLevel))
	config.Log()

	// inject config
	metal_api.Config = config

	server.Run(config.ServerAddress, config.ServerPort)
}

func fetchLogLevel(level string) log.Level {
	level = strings.ToUpper(level)
	for k, v := range logLevels {
		if k == level {
			return v
		}
	}
	return log.WarnLevel
}
