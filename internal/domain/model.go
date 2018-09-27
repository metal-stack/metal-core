package domain

import (
	log "github.com/sirupsen/logrus"
)

type (
	Config struct {
		// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
		LogLevel   string `required:"false" default:"WARN" desc:"set debug level" envconfig:"log_level"`
		ServerPort int    ` required:"false" default:"4242" desc:"set server port" envconfig:"server_port"`
	}
)

func (c Config) Log() {
	log.WithFields(log.Fields{
		"LogLevel":   c.LogLevel,
		"ServerPort": c.ServerPort,
	}).Info("Configuration")
}
