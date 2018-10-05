package api

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"io/ioutil"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	log "github.com/sirupsen/logrus"
)

func (c client) InstallImage(deviceId string) (int, *domain.Device) {
	endpoint := fmt.Sprintf("%s://%s:%d/device/%s/wait", c.Config.APIProtocol, c.Config.APIAddress, c.Config.APIPort, deviceId)

	httpRequest, err := http.NewRequest(http.MethodGet, endpoint, nil)
	httpRequest.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpRequest)
	if err != nil {
		log.Debugf("Install request failed: %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read response from wait call %v", err)
		return resp.StatusCode, nil
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("GET to wait endpoint did not succeed %v: %s", resp.Status, string(body))
		return resp.StatusCode, nil
	}

	dev := &domain.Device{}
	if err = json.Unmarshal(body, dev); err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"body":  string(body),
			"error": err,
		})).Error("Unable to parse json response")
		return resp.StatusCode, nil
	}

	return http.StatusOK, dev
}
