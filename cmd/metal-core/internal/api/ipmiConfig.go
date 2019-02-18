package api

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

func (c *apiClient) IPMIConfig(machineID string) (*domain.IPMIConfig, error) {
	params := machine.NewIPMIDataParams()
	params.ID = machineID

	ok, err := c.MachineClient.IPMIData(params)
	if err != nil {
		zapup.MustRootLogger().Error("IPMI for machine not found",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("IPMI for machine %s not found: %v", machineID, err)
	}

	ipmiData := ok.Payload

	hostAndPort := strings.Split(*ipmiData.Address, ":")
	port, err := strconv.Atoi(hostAndPort[1])
	if err != nil {
		zapup.MustRootLogger().Error("unable to extract port from ipmiaddress",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		port = 632
	}
	ipmiConn := &domain.IPMIConfig{
		Hostname:  hostAndPort[0],
		Port:      port,
		Interface: *ipmiData.Interface,
		Username:  *ipmiData.User,
		Password:  *ipmiData.Password,
	}

	return ipmiConn, nil
}
