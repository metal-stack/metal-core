package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func installEndpoint(w http.ResponseWriter, r *http.Request) {
	devID := mux.Vars(r)["deviceID"]

	log.WithField("deviceID", devID).
		Info("Request metal API for an image to install")

	sc, dev := srv.GetMetalAPIClient().InstallImage(devID)

	logger := log.WithFields(log.Fields{
		"statusCode": sc,
		"deviceId":   devID,
	})

	if sc == http.StatusOK {
		logger.WithFields(log.Fields{
			"imageID":  dev.Image.ID,
			"imageURL": dev.Image.Url,
		}).Info("Got image to install")
		rest.Respond(w, http.StatusOK, dev.Image.Url)
	} else {
		errMsg := "No installation image found"
		logging.Decorate(logger).
			Error(errMsg)
		rest.Respond(w, sc, errMsg)
	}
}
