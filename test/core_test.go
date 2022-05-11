package test

import (
	"errors"
	"net/http"

	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
)

var machineID = "fake-machine-id"

type apiHandlerCoreTest struct{}

func (a *apiHandlerCoreTest) Send(event *v1.EventServiceSendRequest) (*v1.EventServiceSendResponse, error) {
	return nil, nil
}

func (a *apiHandlerCoreTest) FindMachine(machineID string) (*models.V1MachineResponse, error) {
	return nil, nil
}

func (a *apiHandlerCoreTest) FindMachines(mac string) (int, []*models.V1MachineResponse) {
	return -1, nil
}

func (a *apiHandlerCoreTest) RegisterMachine(machineID string, request *domain.MetalHammerRegisterMachineRequest) (int, *models.V1MachineResponse) {
	machine := models.V1MachineResponse{
		ID: &machineID,
	}
	return http.StatusOK, &machine
}

func (a *apiHandlerCoreTest) AbortReinstall(machineID string, request *domain.MetalHammerAbortReinstallRequest) (int, *models.V1BootInfo) {
	return -1, nil
}

func (a *apiHandlerCoreTest) IPMIConfig(machineID string) (*domain.IPMIConfig, error) {
	return nil, nil
}

func (a *apiHandlerCoreTest) FindPartition(id string) (*models.V1PartitionResponse, error) {
	return nil, nil
}

func (a *apiHandlerCoreTest) FinalizeAllocation(machineID, consolePassword string, report *domain.Report) (*machine.FinalizeAllocationOK, error) {
	return nil, nil
}

func (a *apiHandlerCoreTest) RegisterSwitch() (*models.V1SwitchResponse, error) {
	return nil, errors.New("")
}

func (a *apiHandlerCoreTest) ConstantlyPhoneHome() {
}

func (a *apiHandlerCoreTest) SetChassisIdentifyLEDStateOn(machineID, description string) error {
	return nil
}

func (a *apiHandlerCoreTest) SetChassisIdentifyLEDStateOff(machineID, description string) error {
	return nil
}
