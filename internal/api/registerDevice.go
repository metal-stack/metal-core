package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
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
		logging.Decorate(log.WithFields(log.Fields{
			"request": req,
			"error":   err,
		})).Error("Unable to serialize request to json")
		return http.StatusInternalServerError, nil
	}

	httpRequest, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(requestJson))
	httpRequest.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpRequest)
	if err != nil {
		logging.Decorate(log.WithField("error", err)).
			Error("Cannot POST hw json struct to register endpoint")
		return http.StatusInternalServerError, nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logging.Decorate(log.WithField("error", err)).
			Error("Unable to read response from register call")
		return resp.StatusCode, nil
	}

	if resp.StatusCode != http.StatusOK {
		logging.Decorate(log.WithFields(log.Fields{
			"status": resp.Status,
			"body":   string(body),
		})).Error("POST of hardware to register endpoint did not succeed")
		return resp.StatusCode, nil
	}

	dev := &domain.Device{}

	err = json.Unmarshal(body, dev)
	if err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"body":  string(body),
			"error": err,
		})).Error("Unable to parse json response")
		return resp.StatusCode, nil
	}

	return http.StatusOK, dev
}
