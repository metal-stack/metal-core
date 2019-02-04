package api

import (
	"errors"
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

type ipmiDataMock struct {
	simulateError                     bool
	host, port, iface, user, password string
	actualDevID                       string
}

func (m *ipmiDataMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*device.IPMIDataParams)
	m.actualDevID = params.ID
	if m.simulateError {
		return nil, errors.New("not found")
	}
	address := fmt.Sprintf("%v:%v", m.host, m.port)
	return &device.IPMIDataOK{
		Payload: &models.MetalIPMI{
			Address:   &address,
			Interface: &m.iface,
			User:      &m.user,
			Password:  &m.password,
		},
	}, nil
}

func TestIPMIData_OK(t *testing.T) {
	// GIVEN
	m := &ipmiDataMock{
		simulateError: false,
		host:          "1.1.1.1",
		port:          "123",
		iface:         "iface",
		user:          "user",
		password:      "password",
	}

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
	}
	apiClient := NewClient(ctx)

	devID := "fakedeviceID"

	// WHEN
	ipmiConn, err := apiClient.IPMIConfig(devID)

	// THEN
	require.NotNil(t, ipmiConn)
	require.Nil(t, err)
	require.Equal(t, devID, m.actualDevID)
	require.Equal(t, m.host, ipmiConn.Hostname)
	require.Equal(t, m.port, strconv.Itoa(ipmiConn.Port))
	require.Equal(t, m.iface, ipmiConn.Interface)
	require.Equal(t, m.user, ipmiConn.Username)
	require.Equal(t, m.password, ipmiConn.Password)
}

func TestIPMIData_InvalidPort(t *testing.T) {
	// GIVEN
	m := &ipmiDataMock{
		simulateError: false,
		host:          "1.1.1.1",
		port:          "invalidPort",
		iface:         "iface",
		user:          "user",
		password:      "password",
	}

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
	}
	apiClient := NewClient(ctx)

	devID := "fakedeviceID"

	// WHEN
	ipmiConn, err := apiClient.IPMIConfig(devID)

	// THEN
	require.NotNil(t, ipmiConn)
	require.Nil(t, err)
	require.Equal(t, devID, m.actualDevID)
	require.Equal(t, m.host, ipmiConn.Hostname)
	require.Equal(t, 632, ipmiConn.Port)
	require.Equal(t, m.iface, ipmiConn.Interface)
	require.Equal(t, m.user, ipmiConn.Username)
	require.Equal(t, m.password, ipmiConn.Password)
}

func TestIPMIData_Error(t *testing.T) {
	// GIVEN
	m := &ipmiDataMock{
		simulateError: true,
	}

	ctx := &domain.AppContext{
		DeviceClient: device.New(m, strfmt.Default),
	}
	apiClient := NewClient(ctx)

	devID := "fakedeviceID"

	// WHEN
	ipmiConn, err := apiClient.IPMIConfig(devID)

	// THEN
	require.Nil(t, ipmiConn)
	require.NotNil(t, err)
	require.Equal(t, fmt.Sprintf("IPMI for device %s not found: not found", devID), err.Error())
}
