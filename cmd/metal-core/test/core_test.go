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

var devId = "fake-machine-id"

type apiHandlerCoreTest struct{}

func TestLoggingMiddleware(t *testing.T) {
	// given
	e := mockAPIEndpoint(func(ctx *domain.AppContext) domain.APIClient {
		return apiHandlerCoreTest{}
	})
	defer deleteLogFile()

	restful.Add(e.NewMachineService())

	payload := &domain.MetalHammerRegisterMachineRequest{
		UUID: devId,
	}
	payload.Nics = []*models.MetalNic{}
	payload.Disks = []*models.MetalBlockMachine{}

	// when
	sc := doPost(fmt.Sprintf("/machine/register/%v", devId), payload)

	// then
	require.Equal(t, http.StatusOK, sc)
	logs := getLogs()
	require.Contains(t, logs, "Register machine at Metal-API")
	require.Contains(t, logs, "Machine registered")
	require.NotContains(t, logs, "level=error")
}

func (a apiHandlerCoreTest) FindMachines(mac string) (int, []*models.MetalMachine) {
	return -1, nil
}

func (a apiHandlerCoreTest) RegisterMachine(machineId string, request *domain.MetalHammerRegisterMachineRequest) (int, *models.MetalMachine) {
	dev := models.MetalMachine{
		ID: &devId,
	}
	return http.StatusOK, &dev
}

func (a apiHandlerCoreTest) InstallImage(machineId string) (int, *models.MetalMachineWithPhoneHomeToken) {
	return -1, nil
}

func (a apiHandlerCoreTest) IPMIConfig(machineId string) (*domain.IPMIConfig, error) {
	return nil, nil
}
