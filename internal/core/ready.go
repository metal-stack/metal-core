package core

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func readyEndpoint(w http.ResponseWriter, r *http.Request) {
	if body, err := ioutil.ReadAll(r.Body); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Unable to read body")
	} else {
		uuid := mux.Vars(r)["deviceUuid"]

		log.WithFields(log.Fields{
			"deviceUuid": uuid,
			"body":       body,
		}).Info("Inform Metal API about device readiness")

		sc := srv.GetMetalAPIClient().Ready(uuid)

		logger := log.WithFields(log.Fields{
			"deviceUuid": uuid,
			"statusCode": sc,
		})

		if sc != http.StatusOK {
			logger.Error("Device not ready")
		} else {
			logger.Info("Device ready")
		}

		//rest.Respond(w, sc, "")
	}
}
