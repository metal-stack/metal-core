package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"io/ioutil"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func registerEndpoint(w http.ResponseWriter, r *http.Request) {
	if hw, err := ioutil.ReadAll(r.Body); err != nil {
		errMsg := "Unable to read body"
		logging.Decorate(log.WithFields(log.Fields{
			"err": err,
		})).Error(errMsg)

		rest.RespondError(w, http.StatusBadRequest, errMsg)
	} else {
		devID := mux.Vars(r)["deviceID"]

		log.WithFields(log.Fields{
			"deviceID": devID,
			"hardware": string(hw),
		}).Info("Register device at Metal API")

		sc, dev := srv.GetMetalAPIClient().RegisterDevice(devID, hw)

		logger := log.WithFields(log.Fields{
			"deviceID":   devID,
			"statusCode": sc,
			"device":     dev,
		})

		if sc != http.StatusOK {
			errMsg := "Failed to register device"
			logging.Decorate(logger).
				Error(errMsg)
			rest.RespondError(w, sc, errMsg)
		} else {
			logger.Info("Device registered")
			rest.Respond(w, sc, dev)
		}
	}
}
