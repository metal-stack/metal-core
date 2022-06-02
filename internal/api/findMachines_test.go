package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
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
	c := &apiClient{
		machineClient: machine.New(m, strfmt.Default),
	}
	mac := "00:11:22:33:44:55:66:77"

	// WHEN
	sc, _ := c.FindMachines(mac)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, mac, m.actualMAC)
}

func TestFindMachines_Error(t *testing.T) {
	// GIVEN
	m := &searchMachineMock{
		simulateError: true,
	}
	c := &apiClient{
		machineClient: machine.New(m, strfmt.Default),
		log:           zaptest.NewLogger(t),
	}
	mac := "00:11:22:33:44:55:66:77"

	// WHEN
	sc, _ := c.FindMachines(mac)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, mac, m.actualMAC)
}
