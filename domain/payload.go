package domain

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
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
	models.MetalMachineHardware
	UUID string            `json:"uuid,omitempty" description:"the uuid of the machine to register"`
	IPMI *models.MetalIPMI `json:"ipmi" description:"the IPMI connection configuration"`
}
