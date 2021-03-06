package api

import (
	"errors"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/stretchr/testify/require"
)

type finalizeDataMock struct {
	simulateError bool
	machineid     string
	password      string
	primaryDisk   string
	osPartition   string
}

func (m *finalizeDataMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*machine.FinalizeAllocationParams)
	m.machineid = params.ID
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &machine.FinalizeAllocationOK{
		Payload: &models.V1MachineResponse{
			ID:         &m.machineid,
			Allocation: &models.V1MachineAllocation{},
			Hardware: &models.V1MachineHardware{
				Disks: []*models.V1MachineBlockDevice{
					{
						Name: params.Body.Primarydisk,
					},
				},
			},
		},
	}, nil
}

func TestFinalizeAllocation_OK(t *testing.T) {
	// GIVEN
	m := &finalizeDataMock{
		simulateError: false,
		machineid:     "mymachine",
		password:      "x",
		primaryDisk:   "a",
		osPartition:   "b",
	}

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	machineID := "fakemachineID"
	passwd := "password"

	// WHEN
	ok, err := ctx.APIClient().FinalizeAllocation(machineID, passwd, &domain.Report{
		Success:         false,
		Message:         "",
		ConsolePassword: "",
		PrimaryDisk:     "",
		OSPartition:     "",
		Initrd:          "",
		Cmdline:         "",
		Kernel:          "",
		BootloaderID:    "",
	})

	// THEN
	require.Nil(t, err)
	require.Equal(t, machineID, *ok.Payload.ID)
}

func TestFinalizeAllocation_NOK(t *testing.T) {
	// GIVEN
	m := &finalizeDataMock{
		simulateError: true,
	}

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	machineID := "fakemachineID"
	passwd := "password"

	// WHEN
	_, err := ctx.APIClient().FinalizeAllocation(machineID, passwd, &domain.Report{
		Success:         false,
		Message:         "",
		ConsolePassword: "",
		PrimaryDisk:     "",
		OSPartition:     "",
		Initrd:          "",
		Cmdline:         "",
		Kernel:          "",
		BootloaderID:    "",
	})

	// THEN
	require.Error(t, err)
}
