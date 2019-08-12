package test

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
)

var fakeMac = "00:11:22:33:44:55"

type apiHandlerBootTest struct{}

func TestPXEBoot(t *testing.T) {
	// given
	e := mockAPIEndpoint(func(ctx *domain.AppContext) domain.APIClient {
		return &apiHandlerBootTest{}
	})
	defer truncateLogFile()

	restful.Add(e.NewBootService())

	c, _, _ := net.ParseCIDR(cfg.CIDR)
	expected := domain.BootResponse{
		Kernel: "https://blobstore.fi-ts.io/metal/images/metal-hammer/metal-hammer-kernel",
		InitRamDisk: []string{
			"https://blobstore.fi-ts.io/metal/images/metal-hammer/metal-hammer-initrd.img.lz4",
		},
		CommandLine: fmt.Sprintf("METAL_CORE_ADDRESS=%v:%d METAL_API_URL=http://%v:%d", c.String(), cfg.Port, cfg.ApiIP, cfg.ApiPort),
	}

	// when
	bootResponse := &domain.BootResponse{}
	sc := doGet(fmt.Sprintf("/v1/boot/%v", fakeMac), bootResponse)

	// then
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, expected.Kernel, bootResponse.Kernel)
	require.Equal(t, expected.InitRamDisk, bootResponse.InitRamDisk)
	bootResponse.CommandLine = bootResponse.CommandLine[strings.Index(bootResponse.CommandLine, "METAL_CORE_ADDRESS"):]
	require.Equal(t, expected.CommandLine, bootResponse.CommandLine)
}

func (a *apiHandlerBootTest) FindMachines(mac string) (int, []*models.V1MachineResponse) {
	if mac == fakeMac {
		return http.StatusOK, []*models.V1MachineResponse{}
	}
	return http.StatusAlreadyReported, []*models.V1MachineResponse{
		{}, // Simulate at least one existing machine
	}
}

func (a *apiHandlerBootTest) RegisterMachine(machineID string, request *domain.MetalHammerRegisterMachineRequest) (int, *models.V1MachineResponse) {
	return -1, nil
}

func (a *apiHandlerBootTest) InstallImage(machineID string) (int, *models.V1MachineResponse) {
	return -1, nil
}

func (a *apiHandlerBootTest) IPMIConfig(machineID string) (*domain.IPMIConfig, error) {
	return nil, nil
}

func (a *apiHandlerBootTest) FindPartition(id string) (*models.V1PartitionResponse, error) {
	return &models.V1PartitionResponse{Bootconfig: &models.V1PartitionBootConfiguration{
		Kernelurl:   "https://blobstore.fi-ts.io/metal/images/metal-hammer/metal-hammer-kernel",
		Imageurl:    "https://blobstore.fi-ts.io/metal/images/metal-hammer/metal-hammer-initrd.img.lz4",
		Commandline: "",
	}}, nil
}

func (a *apiHandlerBootTest) AddProvisioningEvent(machineID string, event *models.V1MachineProvisioningEvent) error {
	return nil
}

func (a *apiHandlerBootTest) FinalizeAllocation(machineID, consolepassword string) (*machine.FinalizeAllocationOK, error) {
	return nil, nil
}

func (a *apiHandlerBootTest) RegisterSwitch() (*models.V1SwitchResponse, error) {
	return nil, errors.New("")
}

func (a *apiHandlerBootTest) ConstantlyPhoneHome() {
}

func (a *apiHandlerBootTest) SetMachineLEDStateOn(machineID string) error {
	return nil
}

func (a *apiHandlerBootTest) SetMachineLEDStateOff(machineID string) error {
	return nil
}
