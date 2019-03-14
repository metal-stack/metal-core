package api

import (
	"errors"
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
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
	actualmachineID                   string
}

func (m *ipmiDataMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*machine.IPMIDataParams)
	m.actualmachineID = params.ID
	if m.simulateError {
		return nil, errors.New("not found")
	}
	address := fmt.Sprintf("%v:%v", m.host, m.port)
	return &machine.IPMIDataOK{
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
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	machineID := "fakemachineID"

	// WHEN
	ipmiCfg, err := ctx.APIClient().IPMIConfig(machineID)

	// THEN
	require.NotNil(t, ipmiCfg)
	require.Nil(t, err)
	require.Equal(t, machineID, m.actualmachineID)
	require.Equal(t, m.host, ipmiCfg.Hostname)
	require.Equal(t, m.port, strconv.Itoa(ipmiCfg.Port))
	require.Equal(t, m.iface, *ipmiCfg.Ipmi.Interface)
	require.Equal(t, m.user, *ipmiCfg.Ipmi.User)
	require.Equal(t, m.password, *ipmiCfg.Ipmi.Password)
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
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	machineID := "fakemachineID"

	// WHEN
	ipmiCfg, err := ctx.APIClient().IPMIConfig(machineID)

	// THEN
	require.NotNil(t, ipmiCfg)
	require.Nil(t, err)
	require.Equal(t, machineID, m.actualmachineID)
	require.Equal(t, m.host, ipmiCfg.Hostname)
	require.Equal(t, 632, ipmiCfg.Port)
	require.Equal(t, m.iface, *ipmiCfg.Ipmi.Interface)
	require.Equal(t, m.user, *ipmiCfg.Ipmi.User)
	require.Equal(t, m.password, *ipmiCfg.Ipmi.Password)
}

func TestIPMIData_Error(t *testing.T) {
	// GIVEN
	m := &ipmiDataMock{
		simulateError: true,
	}

	ctx := &domain.AppContext{
		MachineClient: machine.New(m, strfmt.Default),
	}
	ctx.SetAPIClient(NewClient)

	machineID := "fakemachineID"

	// WHEN
	ipmiCfg, err := ctx.APIClient().IPMIConfig(machineID)

	// THEN
	require.Nil(t, ipmiCfg)
	require.NotNil(t, err)
	require.Equal(t, fmt.Sprintf("IPMI data for machine %s not found: not found", machineID), err.Error())
}
