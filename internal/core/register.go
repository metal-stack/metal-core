package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func registerEndpoint(w http.ResponseWriter, r *http.Request) {
	if lshw, err := ioutil.ReadAll(r.Body); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Unable to read body")
	} else {
		deviceId := mux.Vars(r)["deviceId"]

		log.WithField("deviceId", deviceId).
			Info("Register device at Metal API")

		statusCode, device := srv.GetMetalAPIClient().RegisterDevice(deviceId, lshw)

		logger := log.WithFields(log.Fields{
			"devideId": deviceId,
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
}
