package api

import (
	"encoding/json"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

func (c client) RegisterDevice(deviceId string, hw []byte) (int, *domain.Device) {
	dev := &domain.Device{}
	rdr := &domain.RegisterDeviceRequest{}
	if err := json.Unmarshal(hw, rdr); err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"hardware": string(hw),
			"error":    err,
		})).Error("Cannot unmarshal request body of hardware")
		return http.StatusBadRequest, nil
	}
	req := domain.MetalApiRegisterDeviceRequest{
		UUID:       deviceId,
		FacilityID: c.GetConfig().FacilityID,
		Hardware: domain.MetalApiDeviceHardware{
			Memory:   rdr.Memory,
			CPUCores: rdr.CPUCores,
			Nics:     rdr.Nics,
			Disks:    rdr.Disks,
		},
	}
	if sc := c.postExpect("/device/register", nil, req, dev); sc != http.StatusOK {
		logging.Decorate(log.WithFields(log.Fields{
			"statusCode": sc,
		})).Error("Failed to POST hardware to Metal-APIs register endpoint")
		return sc, nil
	} else {
		return http.StatusOK, dev
	}
}
