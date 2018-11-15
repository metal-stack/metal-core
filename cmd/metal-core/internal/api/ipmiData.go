package api

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

func (c client) IPMIData(deviceId string) *domain.IpmiConnection {
	params := device.NewIPMIDataParams()
	params.ID = deviceId

	ok, err := c.DeviceClient.IPMIData(params)
	if err != nil {
		zapup.MustRootLogger().Error("Device(s) not found",
			zap.String("mac", deviceId),
			zap.Error(err),
		)
		return nil
	}

	ipmiData := ok.Payload

	hostAndPort := strings.Split(*ipmiData.Address, ":")
	port, err := strconv.Atoi(hostAndPort[0])
	if err != nil {
		port = 632
	}
	ipmiConn := &domain.IpmiConnection{
		Hostname:  hostAndPort[0],
		Port:      port,
		Interface: *ipmiData.Interface,
		Username:  *ipmiData.User,
		Password:  *ipmiData.Password,
	}

	return ipmiConn
}
