package api

import (
	"encoding/json"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

func (c client) RegisterDevice(deviceID string, hw []byte) (int, *domain.Device) {
	rdr := &domain.RegisterDeviceRequest{}
	if err := json.Unmarshal(hw, rdr); err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"hardware": string(hw),
			"error":    err,
		})).Error("Cannot unmarshal request body")
		return http.StatusBadRequest, nil
	}

	req := domain.MetalApiRegisterDeviceRequest{
		UUID:       deviceID,
		FacilityID: c.GetConfig().FacilityID,
		Hardware: domain.MetalApiDeviceHardware{
			Memory:   rdr.Memory,
			CPUCores: rdr.CPUCores,
			Nics:     rdr.Nics,
			Disks:    rdr.Disks,
		},
	}
	dev := &domain.Device{}
	sc := c.postExpect("/device/register", nil, req, dev)
	return sc, dev
}
