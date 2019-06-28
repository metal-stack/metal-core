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

type searchMachineMock struct {
	simulateError bool
	actualMAC     string
}

func (m *searchMachineMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*machine.FindMachinesParams)
	m.actualMAC = params.Body.NicsMacAddresses[0]
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &machine.FindMachinesOK{}, nil
}

func TestFindMachines_OK(t *testing.T) {
	// GIVEN
	m := &searchMachineMock{
		simulateError: false,
	}

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	mac := "00:11:22:33:44:55:66:77"

	// WHEN
	sc, _ := ctx.APIClient().FindMachines(mac)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, mac, m.actualMAC)
}

func TestFindMachines_Error(t *testing.T) {
	// GIVEN
	m := &searchMachineMock{
		simulateError: true,
	}

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	mac := "00:11:22:33:44:55:66:77"

	// WHEN
	sc, _ := ctx.APIClient().FindMachines(mac)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, mac, m.actualMAC)
}
