package domain

import (
	"github.com/metal-stack/metal-core/models"
)

type BootResponse struct {
	Kernel      string   `json:"kernel,omitempty"`
	InitRamDisk []string `json:"initrd"`
	CommandLine string   `json:"cmdline,omitempty"`
}

// Report is send back to metal-core after installation finished
type Report struct {
	Success         bool   `json:"success,omitempty" description:"true if installation succeeded"`
	Message         string `json:"message" description:"if installation failed, the error message"`
	ConsolePassword string `json:"console_password" description:"the console password which was generated while provisioning"`
}

type MetalHammerRegisterMachineRequest struct {
	models.V1MachineHardwareExtended
	UUID string                `json:"uuid,omitempty" description:"the uuid of the machine to register"`
	IPMI *models.V1MachineIPMI `json:"ipmi" description:"the IPMI connection configuration"`
	BIOS *models.V1MachineBIOS `json:"bios" description:"the Bios configuration"`
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
