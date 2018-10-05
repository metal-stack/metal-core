package core

import (
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func installEndpoint(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["deviceId"]

	log.WithField("deviceId", id).
		Info("Request metal API for an image to install")

	sc, device := srv.GetMetalAPIClient().InstallImage(id)

	logger := log.WithFields(log.Fields{
		"statusCode": sc,
		"deviceId":   id,
	})

	if sc == http.StatusOK {
		logger.WithFields(log.Fields{
			"imageID":  device.Image.ID,
			"imageURL": device.Image.Url,
		}).Info("Got image to install")
		//rest.Respond(w, http.StatusOK, image.Url
		w.Write([]byte(device.Image.Url))
	} else {
		logger.Error("No installation image found")
		w.WriteHeader(sc)
		w.Write([]byte("No installation image found"))
	}
}
