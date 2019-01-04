package api

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

func (c client) IPMIData(deviceID string) (*domain.IpmiConnection, error) {
	params := device.NewIPMIDataParams()
	params.ID = deviceID

	ok, err := c.DeviceClient.IPMIData(params)
	if err != nil {
		zapup.MustRootLogger().Error("IPMI for device not found",
			zap.String("device", deviceID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("IPMI for device %s not found: %v", deviceID, err)
	}

	ipmiData := ok.Payload

	hostAndPort := strings.Split(*ipmiData.Address, ":")
	port, err := strconv.Atoi(hostAndPort[1])
	if err != nil {
		zapup.MustRootLogger().Error("unable to extract port from ipmiaddress",
			zap.String("device", deviceID),
			zap.Error(err),
		)
		port = 632
	}
	ipmiConn := &domain.IpmiConnection{
		Hostname:  hostAndPort[0],
		Port:      port,
		Interface: *ipmiData.Interface,
		Username:  *ipmiData.User,
		Password:  *ipmiData.Password,
	}

	return ipmiConn, nil
}
