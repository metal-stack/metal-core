package domain

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type (
	Config struct {
		// Valid log levels are: DEBUG, INFO, WARN, ERROR, FATAL and PANIC
		LogLevel       string `required:"false" default:"INFO" desc:"set log level" split_words:"true"`
		ControlPlaneIP string `required:"true" desc:"set the control plane IP" split_words:"true"`
		FacilityID     string `required:"true" desc:"set the facility ID" split_words:"true"`
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
		ID          string                 `json:"id" description:"a unique ID" unique:"true" readOnly:"true" modelDescription:"A device representing a bare metal machine." rethinkdb:"id,omitempty"`
		Name        string                 `json:"name" description:"the name of the device" rethinkdb:"name"`
		Description string                 `json:"description,omitempty" description:"a description for this machine" optional:"true" rethinkdb:"description"`
		Created     time.Time              `json:"created" description:"the creation time of this machine" optional:"true" readOnly:"true" rethinkdb:"created"`
		Changed     time.Time              `json:"changed" description:"the last changed timestamp" optional:"true" readOnly:"true" rethinkdb:"changed"`
		Project     string                 `json:"project" description:"the project that this device is assigned to" rethinkdb:"project"`
		Facility    Facility               `json:"facility" description:"the facility assigned to this device" readOnly:"true" rethinkdb:"-"`
		FacilityID  string                 `json:"-" rethinkdb:"facilityid"`
		Image       *Image                 `json:"image" description:"the image assigned to this device" readOnly:"true"  rethinkdb:"-"`
		ImageID     string                 `json:"-" rethinkdb:"imageid"`
		Size        *Size                  `json:"size" description:"the size of this device" readOnly:"true" rethinkdb:"-"`
		SizeID      string                 `json:"-" rethinkdb:"sizeid"`
		Hardware    MetalApiDeviceHardware `json:"hardware" description:"the hardware of this device" rethinkdb:"hardware"`
		IP          string                 `json:"ip" description:"the ip address of the allocated device" rethinkdb:"ip"`
		Hostname    string                 `json:"hostname" description:"the hostname of the device" rethinkdb:"hostname"`
		SSHPubKey   string                 `json:"ssh_pub_key" description:"the public ssh key to access the device with" rethinkdb:"sshPubKey"`
	}

	Nic struct {
		MacAddress string   `json:"mac"`
		Name       string   `json:"name"`
		Vendor     string   `json:"vendor"`
		Features   []string `json:"features"`
	}

	BlockDevice struct {
		Name string `json:"name"`
		Size uint64 `json:"size"`
	}

	RegisterDeviceRequest struct {
		UUID     string        `json:"uuid" description:"the uuid of the device to register"`
		Memory   int64         `json:"memory" description:"the memory in bytes of the device to register"`
		CPUCores uint32        `json:"cpucores" description:"the cpu core of the device to register"`
		Nics     []Nic         `json:"nics"`
		Disks    []BlockDevice `json:"disks"`
	}

	MetalApiDeviceHardware struct {
		Memory   int64         `json:"memory" description:"the total memory of the device" rethinkdb:"memory"`
		CPUCores uint32        `json:"cpu_cores" description:"the total memory of the device" rethinkdb:"cpu_cores"`
		Nics     []Nic         `json:"nics" description:"the list of network interfaces of this device" rethinkdb:"network_interfaces"`
		Disks    []BlockDevice `json:"disks" description:"the list of block devices of this device" rethinkdb:"block_devices"`
	}

	MetalApiRegisterDeviceRequest struct {
		UUID       string                 `json:"uuid" description:"the product uuid of the device to register"`
		FacilityID string                 `json:"facilityid" description:"the facility id to register this device with"`
		Hardware   MetalApiDeviceHardware `json:"hardware" description:"the hardware of this device"`
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
		"ID":          d.ID,
		"Name":        d.Name,
		"Description": d.Description,
		"Created":     d.Created,
		"Changed":     d.Changed,
		"Project":     d.Project,
		"FacilityID":  d.Facility.ID,
		"ImageID":     d.Image.ID,
		"SizeID":      d.Size.ID,
	}).Info("Device details")
}
