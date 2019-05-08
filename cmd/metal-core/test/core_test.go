package test

import (
	"fmt"
	"net/http"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
)

var machineID = "fake-machine-id"

type apiHandlerCoreTest struct{}

func TestLoggingMiddleware(t *testing.T) {
	// given
	e := mockAPIEndpoint(func(ctx *domain.AppContext) domain.APIClient {
		return &apiHandlerCoreTest{}
	})
	defer deleteLogFile()

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
	logs := getLogs()
	require.Contains(t, logs, "Register machine at Metal-API")
	require.Contains(t, logs, "Machine registered")
	require.Contains(t, logs, fmt.Sprintf("%q:%q", "id", machineID))
	require.NotContains(t, logs, "level=error")
}

func (a *apiHandlerCoreTest) FindMachines(mac string) (int, []*models.V1MachineListResponse) {
	return -1, nil
}

func (a *apiHandlerCoreTest) RegisterMachine(machineId string, request *domain.MetalHammerRegisterMachineRequest) (int, *models.V1MachineDetailResponse) {
	machine := models.V1MachineDetailResponse{
		ID: &machineID,
	}
	return http.StatusOK, &machine
}

func (a *apiHandlerCoreTest) InstallImage(machineId string) (int, *models.V1MachineWaitResponse) {
	return -1, nil
}

func (a *apiHandlerCoreTest) IPMIConfig(machineId string) (*domain.IPMIConfig, error) {
	return nil, nil
}
func (a *apiHandlerCoreTest) AddProvisioningEvent(machineID string, event *models.V1MachineProvisioningEvent) error {
	return nil
}
