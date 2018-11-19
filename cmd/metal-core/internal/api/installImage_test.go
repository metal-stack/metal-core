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

type installImageMock struct {
	simulateError bool
	actualDevID   string
}

func (m *installImageMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*device.WaitForAllocationParams)
	m.actualDevID = params.ID
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &device.WaitForAllocationOK{}, nil
}

func TestInstallImage_OK(t *testing.T) {
	// GIVEN
	m := &installImageMock{
		simulateError: false,
	}

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
	}
	apiClient := Handler(ctx)

	devId := "fakeDeviceID"

	// WHEN
	sc, _ := apiClient.InstallImage(devId)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, devId, m.actualDevID)
}

func TestInstallImage_Error(t *testing.T) {
	// GIVEN
	m := &installImageMock{
		simulateError: true,
	}

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
	}
	apiClient := Handler(ctx)

	devId := "fakeDeviceID"

	// WHEN
	sc, _ := apiClient.InstallImage(devId)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, devId, m.actualDevID)
}
