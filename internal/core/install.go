package core

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
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
		//rest.Respond(w, http.StatusOK, image.Url)
		rest.Respond(w, http.StatusOK, "https://blobstore.fi-ts.io/metal/images/os/ubuntu/18.04/img.tar.gz")
	} else {
		logger.Error("No installation image found")
		rest.Respond(w, http.StatusNotFound, nil)
	}
}
