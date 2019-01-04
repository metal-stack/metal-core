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

	siteID := "fakeSiteID"
	rackID := "fakeRackID"
	devID := "fakeDeviceID"

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
		Config: &domain.Config{
			SiteID: siteID,
			RackID: rackID,
		},
	}
	apiClient := Handler(ctx)

	payload := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devID,
	}

	// WHEN
	sc, _ := apiClient.RegisterDevice(devID, payload)

	// THEN
	require.Equal(t, http.StatusOK, sc)
	require.Equal(t, devID, m.actualDevID)
	require.Equal(t, siteID, m.actualSiteID)
	require.Equal(t, rackID, m.actualRackID)
}

func TestRegisterDevice_Error(t *testing.T) {
	// GIVEN
	m := &registerDeviceMock{
		simulateError: true,
	}

	siteID := "fakeSiteID"
	rackID := "fakeRackID"
	devID := "fakeDeviceID"

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
		Config: &domain.Config{
			SiteID: siteID,
			RackID: rackID,
		},
	}
	apiClient := Handler(ctx)

	payload := &domain.MetalHammerRegisterDeviceRequest{
		UUID: devID,
	}

	// WHEN
	sc, _ := apiClient.RegisterDevice(devID, payload)

	// THEN
	require.Equal(t, http.StatusInternalServerError, sc)
	require.Equal(t, devID, m.actualDevID)
	require.Equal(t, siteID, m.actualSiteID)
	require.Equal(t, rackID, m.actualRackID)
}
