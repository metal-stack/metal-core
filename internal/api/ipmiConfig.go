package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"go.uber.org/zap"
)

func (c *apiClient) IPMIConfig(machineID string) (*domain.IPMIConfig, error) {
	params := machine.NewFindIPMIMachineParams()
	params.ID = machineID

	ok, err := c.MachineClient.FindIPMIMachine(params, c.Auth)
	if err != nil || ok.Payload == nil || ok.Payload.Ipmi == nil {
		c.Log.Error("ipmi data for machine not found",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("ipmi data for machine %s not found: %w", machineID, err)
	}
	ipmiData := ok.Payload.Ipmi

	if ipmiData.Address == nil {
		c.Log.Error("ipmi address for machine not found",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("ipmi address for machine %s not found: %w", machineID, err)
	}
	hostAndPort := strings.Split(*ipmiData.Address, ":")
	port := 623
	if len(hostAndPort) == 2 {
		port, err = strconv.Atoi(hostAndPort[1])
		if err != nil {
			c.Log.Error("unable to extract port from ipmi address",
				zap.String("machine", machineID),
				zap.String("address", *ipmiData.Address),
				zap.Error(err),
			)
			port = 623
		}
	}

	ipmiCfg := &domain.IPMIConfig{
		Hostname: hostAndPort[0],
		Port:     port,
		Ipmi:     ipmiData,
	}

	return ipmiCfg, nil
}
