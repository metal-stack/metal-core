package core

import (
	"io/ioutil"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func registerEndpoint(w http.ResponseWriter, r *http.Request) {
	hw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Unable to read body")
		return
	}

	log.WithField("register hw", string(hw)).
		Info("Register device")
	deviceID := mux.Vars(r)["deviceId"]

	log.WithField("deviceId", deviceID).
		Info("Register device at Metal API")

	statusCode, device := srv.GetMetalAPIClient().RegisterDevice(deviceID, hw)

	logger := log.WithFields(log.Fields{
		"deviceID":   deviceID,
		"statusCode": statusCode,
		"device":     device,
	})

	if statusCode != http.StatusOK {
		logger.Error("Failed to register device")
	} else {
		logger.Info("Device registered")
	}

	rest.Respond(w, statusCode, device)

}
