package domain

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type (
	Config struct {
		// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
		LogLevel       string `required:"false" default:"INFO" desc:"set log level" envconfig:"log_level"`
		ControlPlaneIP string `required:"false" desc:"set the control plane IP" envconfig:"control_plane_ip"`
		Address        string `required:"false" default:"localhost" desc:"set server address"`
		Port           int    `required:"false" default:"4242" desc:"set server port"`
		APIProtocol    string `required:"false" default:"http" desc:"set metal api protocol" envconfig:"metal_api_protocol"`
		APIAddress     string `required:"false" default:"localhost" desc:"set metal api address" envconfig:"metal_api_address"`
		APIPort        int    `required:"false" default:"8080" desc:"set metal api port" envconfig:"metal_api_port"`
	}
	Facility struct {
		ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"A Facility describes the location where a device is placed."  rethinkdb:"id,omitempty"`
		Name        string    `json:"name" description:"the readable name" rethinkdb:"name"`
		Description string    `json:"description,omitempty" description:"a description for this facility" optional:"true" rethinkdb:"description"`
		Created     time.Time `json:"created" description:"the creation time of this facility" optional:"true" readOnly:"true" rethinkdb:"created"`
		Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	}
	Image struct {
		ID          string    `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device." rethinkdb:"id,omitempty"`
		Name        string    `json:"name" description:"the readable name" rethinkdb:"name"`
		Description string    `json:"description,omitempty" description:"a description for this image" optional:"true" rethinkdb:"description"`
		Url         string    `json:"url" description:"the url to this image" rethinkdb:"url"`
		Created     time.Time `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true" rethinkdb:"created"`
		Changed     time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	}
	Size struct {
		ID          string `json:"id" description:"a unique ID" unique:"true" modelDescription:"An image that can be put on a device." rethinkdb:"id,omitempty"`
		Name        string `json:"name" description:"the readable name" rethinkdb:"name"`
		Description string `json:"description,omitempty" description:"a description for this image" optional:"true" rethinkdb:"description"`
		// Constraints []*Constraint `json:"constraints" description:"a list of constraints that defines this size" optional:"true"`
		Created time.Time `json:"created" description:"the creation time of this image" optional:"true" readOnly:"true" rethinkdb:"created"`
		Changed time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
	}
	Device struct {
		ID           string    `json:"id" description:"a unique ID" unique:"true" readOnly:"true" modelDescription:"A device representing a bare metal machine." rethinkdb:"id,omitempty"`
		Name         string    `json:"name" description:"the name of the device" rethinkdb:"name"`
		Description  string    `json:"description,omitempty" description:"a description for this machine" optional:"true" rethinkdb:"description"`
		Created      time.Time `json:"created" description:"the creation time of this machine" optional:"true" readOnly:"true" rethinkdb:"created"`
		Changed      time.Time `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
		Project      string    `json:"project" description:"the project that this device is assigned to" rethinkdb:"project"`
		Facility     Facility  `json:"facility" description:"the facility assigned to this device" readOnly:"true" rethinkdb:"-"`
		Image        *Image    `json:"image" description:"the image assigned to this device" readOnly:"true"  rethinkdb:"-"`
		Size         Size      `json:"size" description:"the size of this device" readOnly:"true" rethinkdb:"-"`
		FacilityID   string    `json:"-" rethinkdb:"facilityid"`
		ImageID      string    `json:"-" rethinkdb:"imageid"`
		SizeID       string    `json:"-" rethinkdb:"sizeid"`
		MACAddresses []string  `json:"macAddresses" description:"the list of mac addresses in this device" readOnly:"true" rethinkdb:"macAddresses"`
	}
	RegisterDeviceRequest struct {
		ID         string   `json:"id" description:"the id of the device to register"`
		Macs       []string `json:"macs" description:"the mac addresses to register this device with"`
		FacilityID string   `json:"facilityid" description:"the facility id to register this device with"`
		SizeID     string   `json:"sizeid" description:"the size id to register this device with"`
		// Memory     int64  `json:"memory" description:"the size id to assign this device to"`
		// CpuCores   int    `json:"cpucores" description:"the size id to assign this device to"`
	}
	SwitchPort struct {
	}
)

func (c Config) Log() {
	log.WithFields(log.Fields{
		"LogLevel":    c.LogLevel,
		"Address":     c.Address,
		"Port":        c.Port,
		"APIProtocol": c.APIProtocol,
		"APIAddress":  c.APIAddress,
		"APIPort":     c.APIPort,
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
