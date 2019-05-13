package api

import (
	"errors"
	"net/http"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
)

type registerMachineMock struct {
	simulateError                                    bool
	rdr                                              *models.V1MachineRegisterRequest
	actualmachineID, actualPartitionID, actualRackID string
}

func (m *registerMachineMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*machine.RegisterMachineParams)
	m.rdr = params.Body
	m.actualmachineID = *m.rdr.UUID
	m.actualPartitionID = *m.rdr.Partitionid
	m.actualRackID = *m.rdr.Rackid
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &machine.RegisterMachineOK{}, nil
}

func TestRegisterMachine_OK(t *testing.T) {
	// GIVEN
	m := &registerMachineMock{
		simulateError: false,
	}

	partitionID := "fakePartitionID"
	rackID := "fakeRackID"
	machineID := "fakeMachineID"

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
		Config: &domain.Config{
			PartitionID: partitionID,
			RackID:      rackID,
		},
	}
	ctx.SetAPIClient(NewClient)

	payload := &domain.MetalHammerRegisterMachineRequest{
		UUID: machineID,
	}

	// WHEN
	sc, _ := ctx.APIClient().RegisterMachine(machineID, payload)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, machineID, m.actualmachineID)
	require.Equal(t, partitionID, m.actualPartitionID)
	require.Equal(t, rackID, m.actualRackID)
}

func TestRegisterMachine_Error(t *testing.T) {
	// GIVEN
	m := &registerMachineMock{
		simulateError: true,
	}

	partitionID := "fakePartitionID"
	rackID := "fakeRackID"
	machineID := "fakeMachineID"

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
		Config: &domain.Config{
			PartitionID: partitionID,
			RackID:      rackID,
		},
	}
	ctx.SetAPIClient(NewClient)

	payload := &domain.MetalHammerRegisterMachineRequest{
		UUID: machineID,
	}

	// WHEN
	sc, _ := ctx.APIClient().RegisterMachine(machineID, payload)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, machineID, m.actualmachineID)
	require.Equal(t, partitionID, m.actualPartitionID)
	require.Equal(t, rackID, m.actualRackID)
}
