package api

import (
	"errors"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
)

type finalizeDataMock struct {
	simulateError bool
	machineid     string
	password      string
}

func (m *finalizeDataMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*machine.FinalizeAllocationParams)
	m.machineid = params.ID
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &machine.FinalizeAllocationOK{
		Payload: &models.V1MachineResponse{
			ID: &m.machineid,
			Allocation: &models.V1MachineAllocation{
				ConsolePassword: m.password,
			},
		},
	}, nil
}

func TestFinalizeAllocation_OK(t *testing.T) {
	// GIVEN
	m := &finalizeDataMock{
		simulateError: false,
		machineid:     "mymachine",
		password:      "password",
	}

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	machineID := "fakemachineID"
	passwd := "password"

	// WHEN
	ok, err := ctx.APIClient().FinalizeAllocation(machineID, passwd)

	// THEN
	require.Nil(t, err)
	require.Equal(t, machineID, *ok.Payload.ID)
	require.Equal(t, passwd, ok.Payload.Allocation.ConsolePassword)
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
	_, err := ctx.APIClient().FinalizeAllocation(machineID, passwd)

	// THEN
	require.Error(t, err)
}
