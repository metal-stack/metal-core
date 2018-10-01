package metalcore

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func reportDeviceStateEndpoint(w http.ResponseWriter, r *http.Request) {
	if state, err := ioutil.ReadAll(r.Body); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Unable to read body")
	} else {
		deviceUuid := mux.Vars(r)["deviceUuid"]

		log.WithFields(log.Fields{
			"deviceUuid": deviceUuid,
			"state":      state,
		}).Info("Report Metal API about device state")

		statusCode := ApiServer.GetMetalAPIClient().ReportDeviceState(deviceUuid, string(state))

		log.WithFields(log.Fields{
			"statusCode": statusCode,
		}).Info("Device state reported")

		rest.Respond(w, statusCode, nil)
	}
}
