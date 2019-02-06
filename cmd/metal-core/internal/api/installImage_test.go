package api

import (
	"errors"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type installImageMock struct {
	simulateError bool
	actualDevID   string
}

func (m *installImageMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*machine.WaitForAllocationParams)
	m.actualDevID = params.ID
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &machine.WaitForAllocationOK{}, nil
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

	devID := "fakeMachineID"

	// WHEN
	sc, _ := ctx.APIClient().InstallImage(devID)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, devID, m.actualDevID)
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

	devID := "fakeMachineID"

	// WHEN
	sc, _ := ctx.APIClient().InstallImage(devID)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, devID, m.actualDevID)
}
