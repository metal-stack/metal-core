package api

import (
	"strconv"
	"strings"

	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (c *apiClient) IPMIConfig(machineID string) (*domain.IPMIConfig, error) {
	params := machine.NewFindIPMIMachineParams()
	params.ID = machineID

	ok, err := c.MachineClient.FindIPMIMachine(params, c.Auth)
	if err != nil || ok.Payload == nil || ok.Payload.Ipmi == nil {
		zapup.MustRootLogger().Error("IPMI data for machine not found",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return nil, errors.Wrapf(err, "IPMI data for machine %s not found", machineID)
	}
	ipmiData := ok.Payload.Ipmi

	if ipmiData.Address == nil {
		zapup.MustRootLogger().Error("IPMI address for machine not found",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return nil, errors.Wrapf(err, "IPMI address for machine %s not found", machineID)
	}
	hostAndPort := strings.Split(*ipmiData.Address, ":")
	port := 623
	if len(hostAndPort) == 2 {
		port, err = strconv.Atoi(hostAndPort[1])
		if err != nil {
			zapup.MustRootLogger().Error("Unable to extract port from IPMI address",
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
