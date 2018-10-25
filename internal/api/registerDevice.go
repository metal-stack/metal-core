package api

import (
	"encoding/json"
	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

func (c client) RegisterDevice(deviceId string, hw []byte) (int, *models.MetalDevice) {
	rdr := &domain.MetalHammerRegisterDeviceRequest{}
	if err := json.Unmarshal(hw, rdr); err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"hardware": string(hw),
			"error":    err,
		})).Error("Cannot unmarshal request body of hardware")
		return http.StatusBadRequest, nil
	}
	params := device.NewRegisterDeviceParams()
	params.Body = &models.ServiceRegisterRequest{
		UUID:   &deviceId,
		Siteid: &c.GetConfig().SiteID,
		Rackid: &c.GetConfig().RackID,
		Hardware: &models.MetalDeviceHardware{
			Memory:   rdr.Memory,
			CPUCores: rdr.CPUCores,
			Nics:     rdr.Nics,
			Disks:    rdr.Disks,
		},
	}
	if ok, created, err := c.DeviceClient.RegisterDevice(params); err != nil {
		logging.Decorate(log.WithFields(log.Fields{})).
			Error("Failed to POST hardware to Metal-APIs register endpoint")
		return http.StatusInternalServerError, nil
	} else if ok != nil {
		return http.StatusOK, ok.Payload
	} else {
		return http.StatusOK, created.Payload
	}
}
