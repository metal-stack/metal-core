package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
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
	c := &apiClient{
		machineClient: machine.New(m, strfmt.Default),
		partitionID:   partitionID,
		rackID:        rackID,
		log:           zaptest.NewLogger(t),
	}
	payload := &domain.MetalHammerRegisterMachineRequest{
		UUID: machineID,
	}

	// WHEN
	sc, _ := c.RegisterMachine(machineID, payload)

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
	c := &apiClient{
		machineClient: machine.New(m, strfmt.Default),
		partitionID:   partitionID,
		rackID:        rackID,
		log:           zaptest.NewLogger(t),
	}
	payload := &domain.MetalHammerRegisterMachineRequest{
		UUID: machineID,
	}

	// WHEN
	sc, _ := c.RegisterMachine(machineID, payload)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, machineID, m.actualmachineID)
	require.Equal(t, partitionID, m.actualPartitionID)
	require.Equal(t, rackID, m.actualRackID)
}
