package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

func (c client) InstallImage(deviceId string) (int, *domain.Device) {
	endpoint := fmt.Sprintf("%s://%s:%d/device/%s/wait", c.Config.APIProtocol, c.Config.APIAddress, c.Config.APIPort, deviceId)

	httpRequest, err := http.NewRequest(http.MethodGet, endpoint, nil)
	httpRequest.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	log.Infof("Starting long polling for device %s", deviceId)
	for {
		client := &http.Client{}
		resp, err = client.Do(httpRequest)
		if err != nil {
			log.Debugf("Long poll request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Errorf("GET to wait endpoint did not succeed %v: %s", resp.Status, string(body))
			return resp.StatusCode, nil
		} else {
			break
		}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read response from wait call %v", err)
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
