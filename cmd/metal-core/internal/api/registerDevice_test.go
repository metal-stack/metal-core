package api

import (
	"errors"
	"net/http"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
)

type registerDeviceMock struct {
	simulateError                           bool
	rdr                                     *models.MetalRegisterDevice
	actualDevID, actualPartitionID, actualRackID string
}

func (m *registerDeviceMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*device.RegisterDeviceParams)
	m.rdr = params.Body
	m.actualDevID = *m.rdr.UUID
	m.actualPartitionID = *m.rdr.Partitionid
	m.actualRackID = *m.rdr.Rackid
	if m.simulateError {
		return nil, errors.New("not found")
	}
	return &device.RegisterDeviceOK{}, nil
}

func TestRegisterDevice_OK(t *testing.T) {
	// GIVEN
	m := &registerDeviceMock{
		simulateError: false,
	}

	partitionID := "fakePartitionID"
	rackID := "fakeRackID"
	devID := "fakeDeviceID"

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
		Config: &domain.Config{
			PartitionID: partitionID,
			RackID: rackID,
		},
	}
	ctx.SetAPIClient(NewClient)

	payload := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devID,
	}

	// WHEN
	sc, _ := ctx.APIClient().RegisterDevice(devID, payload)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, devID, m.actualDevID)
	require.Equal(t, partitionID, m.actualPartitionID)
	require.Equal(t, rackID, m.actualRackID)
}

func TestRegisterDevice_Error(t *testing.T) {
	// GIVEN
	m := &registerDeviceMock{
		simulateError: true,
	}

	partitionID := "fakePartitionID"
	rackID := "fakeRackID"
	devID := "fakeDeviceID"

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
		Config: &domain.Config{
			PartitionID: partitionID,
			RackID: rackID,
		},
	}
	ctx.SetAPIClient(NewClient)

	payload := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devID,
	}

	// WHEN
	sc, _ := ctx.APIClient().RegisterDevice(devID, payload)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, devID, m.actualDevID)
	require.Equal(t, partitionID, m.actualPartitionID)
	require.Equal(t, rackID, m.actualRackID)
}
