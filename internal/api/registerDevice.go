package api

import (
	"encoding/json"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type registerDeviceRequest struct {
	ID         string   `json:"id" description:"the id of the device to register"`
	Macs       []string `json:"macs" description:"the mac addresses to register this device with"`
	FacilityID string   `json:"facilityid" description:"the facility id to register this device with"`
	SizeID     string   `json:"sizeid" description:"the size id to register this device with"`
	// Memory     int64  `json:"memory" description:"the size id to assign this device to"`
	// CpuCores   int    `json:"cpucores" description:"the size id to assign this device to"`
}

func (c client) RegisterDevice(deviceId string, hw []byte) (int, *domain.Device) {
	rdr := domain.RegisterDeviceRequest{}
	if err := json.Unmarshal(hw, rdr); err != nil {
		log.Error("Cannot unmarshal request body")
		return http.StatusBadRequest, nil
	}
	macs := make([]string, len(rdr.Nics))
	for i := range rdr.Nics {
		macs[i] = rdr.Nics[i].MacAddress
	}
	sizeId := "t1.small.x86"
	//TODO Fetch sizeId from Metal-API by providing rdr.Memory and rdr.CPUCores values
	req := registerDeviceRequest{
		ID:         deviceId,
		Macs:       macs,
		FacilityID: c.GetConfig().FacilityID,
		SizeID:     sizeId,
	}
	var dev *domain.Device
	sc := c.postExpect("/device/register", nil, req, dev)
	return sc, dev
}
