package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func readyEndpoint(w http.ResponseWriter, r *http.Request) {
	if body, err := ioutil.ReadAll(r.Body); err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"err": err,
		})).Error("Unable to read body")
	} else {
		devId := mux.Vars(r)["deviceId"]

		log.WithFields(log.Fields{
			"deviceID": devId,
			"body":     body,
		}).Info("Inform Metal API about device readiness")

		sc := srv.GetMetalAPIClient().Ready(devId)

		logger := log.WithFields(log.Fields{
			"deviceID":   devId,
			"statusCode": sc,
		})

		if sc != http.StatusOK {
			logging.Decorate(logger).
				Error("Device not ready")
		} else {
			logger.Info("Device ready")
		}

		//rest.Respond(w, sc, "")
	}
}
