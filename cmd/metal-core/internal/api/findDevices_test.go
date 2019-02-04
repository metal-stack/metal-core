package api

import (
	"errors"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type searchDeviceMock struct {
	simulateError bool
	actualMAC     *string
}

func (m *searchDeviceMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*device.SearchDeviceParams)
	m.actualMAC = params.Mac
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &device.SearchDeviceOK{}, nil
}

func TestFindDevices_OK(t *testing.T) {
	// GIVEN
	m := &searchDeviceMock{
		simulateError: false,
	}

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	mac := "00:11:22:33:44:55:66:77"

	// WHEN
	sc, _ := ctx.APIClient().FindDevices(mac)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, &mac, m.actualMAC)
}

func TestFindDevices_Error(t *testing.T) {
	// GIVEN
	m := &searchDeviceMock{
		simulateError: true,
	}

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	mac := "00:11:22:33:44:55:66:77"

	// WHEN
	sc, _ := ctx.APIClient().FindDevices(mac)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, &mac, m.actualMAC)
}
