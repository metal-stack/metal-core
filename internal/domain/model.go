package domain

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type (
	Config struct {
		// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
		LogLevel         string `required:"false" default:"WARN" desc:"set log level" envconfig:"log_level"`
		ServerAddress    string `required:"false" default:"localhost" desc:"set server address" envconfig:"server_address"`
		ServerPort       int    `required:"false" default:"4242" desc:"set server port" envconfig:"server_port"`
		MetalApiProtocol string `required:"false" default:"http" desc:"set metal api protocol" envconfig:"metal_api_protocol"`
		MetalApiAddress  string `required:"false" default:"localhost" desc:"set metal api address" envconfig:"metal_api_address"`
		MetalApiPort     int    `required:"false" default:"8080" desc:"set metal api port" envconfig:"metal_api_port"`
	}
	Facility struct {
		ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"A Facility describes the location where a device is placed."`
		Name        string    `json:"name" description:"the readable name"`
		Description string    `json:"description,omitempty" description:"a description for this facility" optional:"true"`
		Created     time.Time `json:"created" description:"the creation time of this facility" optional:"true" readOnly:"true"`
		Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true"`
	}
	Image struct {
		ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device."`
		Name        string    `json:"name" description:"the readable name"`
		Description string    `json:"description,omitempty" description:"a description for this image" optional:"true"`
		Url         string    `json:"url" description:"the url to this image"`
		Created     time.Time `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true"`
		Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true"`
	}
	Size struct {
		ID          string `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device."`
		Name        string `json:"name" description:"the readable name"`
		Description string `json:"description,omitempty" description:"a description for this image" optional:"true"`
		// Constraints []*Constraint `json:"constraints" description:"a list of constraints that defines this size" optional:"true"`
		Created time.Time `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true"`
		Changed time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true"`
	}
	Device struct {
		ID           string    `json:"id" description:"a unique ID" unique:"true" readOnly:"true" modelDescription:"A device representing a bare metal machine."`
		Name         string    `json:"name" description:"the name of the device"`
		Description  string    `json:"description,omitempty" description:"a description for this machine" optional:"true"`
		Created      time.Time `json:"created" description:"the creation time of this machine" optional:"true" readOnly:"true"`
		Changed      time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true"`
		Project      string    `json:"project" description:"the project that this device is assigned to"`
		Facility     Facility  `json:"facility" description:"the facility assigned to this device" readOnly:"true"`
		Image        Image     `json:"image" description:"the image assigned to this device" readOnly:"true"`
		Size         Size      `json:"size" description:"the size of this device" readOnly:"true"`
		MACAddresses []string  `json:"macAddresses" description:"the list of mac addresses in this device" readOnly:"true"`
	}
	RegisterDeviceRequest struct {
		UUID       string   `json:"uuid" description:"the uuid of the device to register"`
		Macs       []string `json:"macs" description:"the mac addresses to register this device with"`
		FacilityID string   `json:"facilityid" description:"the facility id to register this device with"`
		SizeID     string   `json:"sizeid" description:"the size id to register this device with"`
		// Memory     int64  `json:"memory" description:"the size id to assign this device to"`
		// CpuCores   int    `json:"cpucores" description:"the size id to assign this device to"`
	}
)

type (
	MetalcoreAPIServer interface {
		GetMetalAPIClient() MetalAPIClient
		GetConfig() Config
		Run()
	}
	MetalAPIClient interface {
		GetConfig() Config
		FindDevices(mac string) (int, []Device)
		RegisterDevice(lshw string) (int, Device)
		ReportDeviceState(deviceUuid string, state string) int
	}
)

func (c Config) Log() {
	log.WithFields(log.Fields{
		"LogLevel":         c.LogLevel,
		"ServerAddress":    c.ServerAddress,
		"ServerPort":       c.ServerPort,
		"MetalApiProtocol": c.MetalApiProtocol,
		"MetalApiAddress":  c.MetalApiAddress,
		"MetalApiPort":     c.MetalApiPort,
	}).Info("Configuration")
}

func (d Device) Log() {
	log.WithFields(log.Fields{
		"ID":           d.ID,
		"Name":         d.Name,
		"Description":  d.Description,
		"Created":      d.Created,
		"Changed":      d.Changed,
		"Project":      d.Project,
		"FacilityID":   d.Facility.ID,
		"ImageID":      d.Image.ID,
		"SizeID":       d.Size.ID,
		"MACAddresses": strings.Join(d.MACAddresses, ", "),
	}).Info("Device details")
}
