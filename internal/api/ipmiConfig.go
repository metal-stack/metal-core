package api

import (
	"strconv"
	"strings"

	"github.com/metal-stack/metal-core/client/machine"
	"github.com/metal-stack/metal-core/pkg/domain"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (c *apiClient) IPMIConfig(machineID string) (*domain.IPMIConfig, error) {
	params := machine.NewFindIPMIMachineParams()
	params.ID = machineID

	ok, err := c.MachineClient.FindIPMIMachine(params, c.Auth)
	if err != nil {
		zapup.MustRootLogger().Error("IPMI data for machine not found",
			zap.String("machine", machineID),
			zap.Error(err),
		)
		return nil, errors.Wrapf(err, "IPMI data for machine %s not found", machineID)
	}

	ipmi := ok.Payload
	if ipmi.IPMI == nil {
		zapup.MustRootLogger().Error("IPMI data for machine is nil",
			zap.String("machine", machineID),
		)
		return nil, errors.Wrapf(err, "IPMI data for machine %s is nil", machineID)
	}

	hostAndPort := strings.Split(*ipmi.IPMI.Address, ":")
	port := 632
	if len(hostAndPort) == 2 {
		port, err = strconv.Atoi(hostAndPort[1])
		if err != nil {
			zapup.MustRootLogger().Error("Unable to extract port from IPMI address",
				zap.String("machine", machineID),
				zap.String("address", *ipmi.IPMI.Address),
				zap.Error(err),
			)
			port = 632
		}
	}

	ipmiCfg := &domain.IPMIConfig{
		Hostname: hostAndPort[0],
		Port:     port,
		Ipmi:     ipmi.IPMI,
	}

	return ipmiCfg, nil
}
