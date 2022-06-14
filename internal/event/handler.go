package event

import (
	"go.uber.org/zap"
)

type EventHandler struct {
	log *zap.Logger
}

func NewHandler(log *zap.Logger) *EventHandler {
	return &EventHandler{
		log: log,
	}
}

type MachineEvent struct {
	Type         EventType           `json:"type,omitempty"`
	OldMachineID string              `json:"old,omitempty"`
	Cmd          *MachineExecCommand `json:"cmd,omitempty"`
	IPMI         *IPMI               `json:"ipmi,omitempty"`
}

type MachineExecCommand struct {
	TargetMachineID string         `json:"target,omitempty"`
	Command         MachineCommand `json:"cmd,omitempty"`
	Params          []string       `json:"params,omitempty"`
}

type IPMI struct {
	// Address is host:port of the connection to the ipmi BMC, host can be either a ip address or a hostname
	Address  string `json:"address"`
	User     string `json:"user"`
	Password string `json:"password"`
	Fru      Fru    `json:"fru"`
}

type Fru struct {
	BoardPartNumber string `json:"board_part_number"`
}

type EventType string

type MachineCommand string

const (
	MachineOnCmd             MachineCommand = "ON"
	MachineOffCmd            MachineCommand = "OFF"
	MachineResetCmd          MachineCommand = "RESET"
	MachineCycleCmd          MachineCommand = "CYCLE"
	MachineBiosCmd           MachineCommand = "BIOS"
	MachineDiskCmd           MachineCommand = "DISK"
	MachinePxeCmd            MachineCommand = "PXE"
	MachineReinstallCmd      MachineCommand = "REINSTALL"
	ChassisIdentifyLEDOnCmd  MachineCommand = "LED-ON"
	ChassisIdentifyLEDOffCmd MachineCommand = "LED-OFF"
	UpdateFirmwareCmd        MachineCommand = "UPDATE-FIRMWARE"
)

// Some EventType enums.
const (
	Create  EventType = "create"
	Update  EventType = "update"
	Delete  EventType = "delete"
	Command EventType = "command"
)
