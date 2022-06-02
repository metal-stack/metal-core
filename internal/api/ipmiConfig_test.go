package api

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type ipmiDataMock struct {
	simulateError                     bool
	host, port, iface, user, password string
	actualmachineID                   string
}

func (m *ipmiDataMock) Submit(o *runtime.ClientOperation) (interface{}, error) {
	params := o.Params.(*machine.FindIPMIMachineParams)
	m.actualmachineID = params.ID
	if m.simulateError {
		return nil, errors.New("not found")
	}
	address := fmt.Sprintf("%v:%v", m.host, m.port)
	return &machine.FindIPMIMachineOK{
		Payload: &models.V1MachineIPMIResponse{
			Ipmi: &models.V1MachineIPMI{
				Address:   &address,
				Interface: &m.iface,
				User:      &m.user,
				Password:  &m.password,
			},
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
	c := &apiClient{
		machineClient: machine.New(m, strfmt.Default),
		log:           zaptest.NewLogger(t),
	}
	machineID := "fakemachineID"

	// WHEN
	ipmiCfg, err := c.IPMIConfig(machineID)

	// THEN
	require.NotNil(t, ipmiCfg)
	require.Nil(t, err)
	require.Equal(t, machineID, m.actualmachineID)
	require.Equal(t, m.host, ipmiCfg.Hostname)
	require.Equal(t, m.port, strconv.Itoa(ipmiCfg.Port))
	require.Equal(t, m.iface, ipmiCfg.Interface())
	require.Equal(t, m.user, ipmiCfg.User())
	require.Equal(t, m.password, ipmiCfg.Password())
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
	c := &apiClient{
		machineClient: machine.New(m, strfmt.Default),
		log:           zaptest.NewLogger(t),
	}
	machineID := "fakemachineID"

	// WHEN
	ipmiCfg, err := c.IPMIConfig(machineID)

	// THEN
	require.NotNil(t, ipmiCfg)
	require.Nil(t, err)
	require.Equal(t, machineID, m.actualmachineID)
	require.Equal(t, m.host, ipmiCfg.Hostname)
	require.Equal(t, 623, ipmiCfg.Port)
	require.Equal(t, m.iface, ipmiCfg.Interface())
	require.Equal(t, m.user, ipmiCfg.User())
	require.Equal(t, m.password, ipmiCfg.Password())
}

func TestIPMIData_Error(t *testing.T) {
	// GIVEN
	m := &ipmiDataMock{
		simulateError: true,
	}
	c := &apiClient{
		machineClient: machine.New(m, strfmt.Default),
		log:           zaptest.NewLogger(t),
	}
	machineID := "fakemachineID"

	// WHEN
	ipmiCfg, err := c.IPMIConfig(machineID)

	// THEN
	require.Nil(t, ipmiCfg)
	require.NotNil(t, err)
	require.Equal(t, fmt.Sprintf("ipmi data for machine %s not found: not found", machineID), err.Error())
}
