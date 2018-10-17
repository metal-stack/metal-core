package domain

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	log "github.com/sirupsen/logrus"
)

type (
	Config struct {
		// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
		LogLevel          string `required:"false" default:"INFO" desc:"set log level" split_words:"true"`
		ControlPlaneIP    string `required:"true" desc:"set the control plane IP" split_words:"true"`
		FacilityID        string `required:"true" desc:"set the facility ID" split_words:"true"`
		Address           string `required:"false" default:"localhost" desc:"set server address"`
		Port              int    `required:"false" default:"4242" desc:"set server port"`
		APIProtocol       string `required:"false" default:"http" desc:"set metal api protocol" envconfig:"metal_api_protocol"`
		APIAddress        string `required:"false" default:"localhost" desc:"set metal api address" envconfig:"metal_api_address"`
		APIPort           int    `required:"false" default:"8080" desc:"set metal api port" envconfig:"metal_api_port"`
		HammerImagePrefix string `required:"false" default:"pxeboot" desc:"set hammer image prefix for kernel, initrd and cmdline download" split_words:"true"`
	}
	MetalHammerRegisterDeviceRequest struct {
		models.MetalDeviceHardware
		UUID string `json:"uuid" description:"the uuid of the device to register"`
	}
)

func (c Config) Log() {
	log.WithFields(log.Fields{
		"LogLevel":          c.LogLevel,
		"Address":           c.Address,
		"Port":              c.Port,
		"APIProtocol":       c.APIProtocol,
		"APIAddress":        c.APIAddress,
		"APIPort":           c.APIPort,
		"HammerImagePrefix": c.HammerImagePrefix,
	}).Info("Configuration")
}
