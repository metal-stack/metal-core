package api

import (
	"encoding/json"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

type registerDeviceRequest struct {
	UUID       string               `json:"uuid" description:"the product uuid of the device to register"`
	FacilityID string               `json:"facilityid" description:"the facility id to register this device with"`
	Hardware   metal.DeviceHardware `json:"hardware" description:"the hardware of this device"`
}

func (c client) RegisterDevice(deviceId string, hw []byte) (int, *domain.Device) {
	rdr := domain.RegisterDeviceRequest{}
	if err := json.Unmarshal(hw, rdr); err != nil {
		log.Error("Cannot unmarshal request body")
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
