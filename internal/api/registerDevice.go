package api

import (
	"encoding/json"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

func (c client) RegisterDevice(deviceId string, hw []byte) (int, *domain.Device) {
	rdr := &domain.RegisterDeviceRequest{}
	if err := json.Unmarshal(hw, rdr); err != nil {
		log.Errorf("Cannot unmarshal request body of hw:%s error:%v", string(hw), err)
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
	var dev *domain.Device
	sc := c.postExpect("/device/register", nil, req, dev)
	return sc, dev
}
