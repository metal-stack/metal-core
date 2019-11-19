package api

import (
	"errors"
	"net/http"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"git.f-i-ts.de/cloud-native/metal/metal-core/pkg/domain"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
)

type installImageMock struct {
	simulateError   bool
	actualmachineID string
}

func (m *installImageMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*machine.WaitForAllocationParams)
	m.actualmachineID = params.ID
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &machine.WaitForAllocationOK{Payload: &models.V1MachineResponse{
		Allocation: &models.V1MachineAllocation{Image: &models.V1ImageResponse{}}},
	}, nil
}

func TestInstallImage_OK(t *testing.T) {
	// GIVEN
	m := &installImageMock{
		simulateError: false,
	}

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	machineID := "fakeMachineID"

	// WHEN
	sc, _ := ctx.APIClient().InstallImage(machineID)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, machineID, m.actualmachineID)
}

func TestInstallImage_Error(t *testing.T) {
	// GIVEN
	m := &installImageMock{
		simulateError: true,
	}

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	machineID := "fakeMachineID"

	// WHEN
	sc, _ := ctx.APIClient().InstallImage(machineID)

	// THEN
	require.Equal(t, http.StatusNotModified, sc)
	require.Equal(t, machineID, m.actualmachineID)
}
