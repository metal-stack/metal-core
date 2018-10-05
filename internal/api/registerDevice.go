package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	endpoint := fmt.Sprintf("%s://%s:%d/device/register", c.Config.APIProtocol, c.Config.APIAddress, c.Config.APIPort)

	requestJson, err := json.Marshal(req)
	if err != nil {
		log.Errorf("unable to serialize request %v to json %v", req, err)
		return http.StatusInternalServerError, nil
	}

	httpRequest, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(requestJson))
	httpRequest.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpRequest)
	if err != nil {
		log.Errorf("cannot POST hw json struct to register endpoint: %v", err)
		return http.StatusInternalServerError, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read response from register call %v", err)
		return resp.StatusCode, nil
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("POST of hw to register endpoint did not succeed %v", resp.Status)
		return resp.StatusCode, nil
	}

	var device domain.Device

	err = json.Unmarshal(body, &device)
	if err != nil {
		log.Errorf("Unable to parse json response: %v: %v", body, err)
		return resp.StatusCode, nil
	}

	return http.StatusOK, &device
}
