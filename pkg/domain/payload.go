package domain

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
)

type BootResponse struct {
	Kernel      string   `json:"kernel,omitempty"`
	InitRamDisk []string `json:"initrd"`
	CommandLine string   `json:"cmdline,omitempty"`
}

type Reboot struct {
	HD   bool `json:"hd,omitempty" description:"whether to boot from Disk"`
	PXE  bool `json:"pxe,omitempty" description:"whether to boot from PXE"`
	BIOS bool `json:"bios,omitempty" description:"whether to boot into BIOS"`
}

// Report is sent by metal-hammer to metal-core after installation finished
type Report struct {
	Success         bool   `json:"success,omitempty" description:"true if installation succeeded"`
	Message         string `json:"message" description:"if installation failed, the error message"`
	ConsolePassword string `json:"console_password" description:"the console password which was generated while provisioning"`
	PrimaryDisk     string `json:"primary_disk" description:"the disk having a partition on which the OS is installed"`
	OSPartition     string `json:"os_partition" description:"the partition on which the OS is installed"`
}

type MetalHammerRegisterMachineRequest struct {
	models.V1MachineHardwareExtended
	UUID string                `json:"uuid,omitempty" description:"the uuid of the machine to register"`
	IPMI *models.V1MachineIPMI `json:"ipmi" description:"the IPMI connection configuration"`
}

func (r *MetalHammerRegisterMachineRequest) IPMIAddress() string {
	return IPMIAddress(r.IPMI)
}

func (r *MetalHammerRegisterMachineRequest) IPMIInterface() string {
	return IPMIInterface(r.IPMI)
}

func (r *MetalHammerRegisterMachineRequest) IPMIMAC() string {
	return IPMIMAC(r.IPMI)
}

func (r *MetalHammerRegisterMachineRequest) IPMIUser() string {
	return IPMIUser(r.IPMI)
}

func (r *MetalHammerRegisterMachineRequest) IPMIPassword() string {
	return IPMIPassword(r.IPMI)
}
