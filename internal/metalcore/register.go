package metalcore

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
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
		deviceUuid := mux.Vars(r)["deviceUuid"]

		log.WithField("deviceUuid", deviceUuid).
			Info("Register device at Metal API")

		statusCode, device := srv.GetMetalAPIClient().RegisterDevice(deviceUuid, lshw)

		logger := log.WithFields(log.Fields{
			"devideUuid": deviceUuid,
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
