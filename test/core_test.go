package test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/emicklei/go-restful/v3"
	v1 "github.com/metal-stack/metal-api/pkg/api/v1"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/stretchr/testify/require"
)

var machineID = "fake-machine-id"

type apiHandlerCoreTest struct{}

func TestLoggingMiddleware(t *testing.T) {
	// given
	e := mockAPIEndpoint(func(ctx *domain.AppContext) domain.APIClient {
		return &apiHandlerCoreTest{}
	})

	restful.Add(e.NewMachineService())

	payload := &domain.MetalHammerRegisterMachineRequest{
		UUID: machineID,
	}
	payload.Nics = []*models.V1MachineNic{}
	payload.Disks = []*models.V1MachineBlockDevice{}

	// when
	sc := doPost(fmt.Sprintf("/machine/register/%v", machineID), payload)

	// then
	require.Equal(t, http.StatusOK, sc)
}

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
