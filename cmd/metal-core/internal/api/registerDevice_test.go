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
	actualDevID, actualSiteID, actualRackID string
}

func (m *registerDeviceMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*device.RegisterDeviceParams)
	m.rdr = params.Body
	m.actualDevID = *m.rdr.UUID
	m.actualSiteID = *m.rdr.Siteid
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

	siteId := "fakeSiteID"
	rackId := "fakeRackID"
	devId := "fakeDeviceID"

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
		Config: &domain.Config{
			SiteID: siteId,
			RackID: rackId,
		},
	}
	apiClient := Handler(ctx)

	payload := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devId,
	}

	// WHEN
	sc, _ := apiClient.RegisterDevice(devId, payload)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, devId, m.actualDevID)
	require.Equal(t, siteId, m.actualSiteID)
	require.Equal(t, rackId, m.actualRackID)
}

func TestRegisterDevice_Error(t *testing.T) {
	// GIVEN
	m := &registerDeviceMock{
		simulateError: true,
	}

	siteId := "fakeSiteID"
	rackId := "fakeRackID"
	devId := "fakeDeviceID"

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
		Config: &domain.Config{
			SiteID: siteId,
			RackID: rackId,
		},
	}
	apiClient := Handler(ctx)

	payload := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devId,
	}

	// WHEN
	sc, _ := apiClient.RegisterDevice(devId, payload)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, devId, m.actualDevID)
	require.Equal(t, siteId, m.actualSiteID)
	require.Equal(t, rackId, m.actualRackID)
}
