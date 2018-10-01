package metalcore

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
		deviceUuid := mux.Vars(r)["deviceUuid"]

		log.WithFields(log.Fields{
			"deviceUuid": deviceUuid,
			"body":       body,
		}).Info("Inform Metal API about device readiness")

		statusCode := srv.GetMetalAPIClient().Ready(deviceUuid)

		logger := log.WithFields(log.Fields{
			"deviceUuid": deviceUuid,
			"statusCode": statusCode,
		})

		if statusCode != http.StatusOK {
			logger.Error("Device not ready")
		} else {
			logger.Info("Device ready")
		}

		//rest.Respond(w, statusCode, "")
	}
}
