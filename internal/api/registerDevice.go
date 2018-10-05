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
	var nics []metal.Nic
	for _, nic := range rdr.Nics {
		nics = append(nics, metal.Nic{
			MacAddress: nic.MacAddress,
			Name:       nic.Name,
			Vendor:     nic.Vendor,
			Features:   nic.Features,
		})
	}
	var disks []metal.BlockDevice
	for _, disk := range rdr.Disks {
		disks = append(disks, metal.BlockDevice{
			Size: disk.Size,
			Name: disk.Name,
		})
	}
	req := registerDeviceRequest{
		UUID:       deviceId,
		FacilityID: c.GetConfig().FacilityID,
		Hardware: metal.DeviceHardware{
			Memory:   rdr.Memory,
			CPUCores: rdr.CPUCores,
			Nics:     nics,
			Disks:    disks,
		},
	}
	var dev *domain.Device
	sc := c.postExpect("/device/register", nil, req, dev)
	return sc, dev
}
