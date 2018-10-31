package domain

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	log "github.com/sirupsen/logrus"
)

type (
	Config struct {
		// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
		IP                string `required:"true" desc:"set the metal core IP"`
		SiteID            string `required:"true" desc:"set the site ID" split_words:"true"`
		RackID            string `required:"true" desc:"set the rack ID" split_words:"true"`
		BindAddress       string `required:"false" default:"0.0.0.0" desc:"set server bind address" split_words:"true"`
		Port              int    `required:"false" default:"4242" desc:"set server port"`
		LogLevel          string `required:"false" default:"INFO" desc:"set log level" split_words:"true"`
		ApiProtocol       string `required:"false" default:"http" desc:"set metal api protocol" envconfig:"metal_api_protocol"`
		ApiIP             string `required:"false" default:"localhost" desc:"set metal api address" envconfig:"metal_api_ip"`
		ApiPort           int    `required:"false" default:"8080" desc:"set metal api port" envconfig:"metal_api_port"`
		HammerImagePrefix string `required:"false" default:"pxeboot" desc:"set hammer image prefix for kernel, initrd and cmdline download" split_words:"true"`
		MQAddress         string `required:"false" default:"localhost:4161" desc:"set the MQ server address" envconfig:"mq_address"`
	}

	EventType string

	MetalHammerRegisterDeviceRequest struct {
		models.MetalDeviceHardware
		UUID string `json:"uuid" description:"the uuid of the device to register"`
	}

	DeviceEvent struct {
		Type EventType           `json:"type,omitempty"`
		Old  *models.MetalDevice `json:"old,omitempty"`
		New  *models.MetalDevice `json:"new,omitempty"`
	}
)

// Some EventType enums.
const (
	CREATE EventType = "create"
	UPDATE EventType = "update"
	DELETE EventType = "delete"
)

func (c Config) Log() {
	log.WithFields(log.Fields{
		"LogLevel":          c.LogLevel,
		"BindAddress":       c.BindAddress,
		"IP":                c.IP,
		"Port":              c.Port,
		"API-Protocol":      c.ApiProtocol,
		"API-IP":            c.ApiIP,
		"API-Port":          c.ApiPort,
		"HammerImagePrefix": c.HammerImagePrefix,
	}).Info("Configuration")
}
