package metalcore

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func installEndpoint(w http.ResponseWriter, r *http.Request) {
	deviceUuid := mux.Vars(r)["deviceUuid"]

	log.WithField("deviceUuid", deviceUuid).
		Info("Request metal API for an image to install")

	statusCode, image := srv.GetMetalAPIClient().InstallImage(deviceUuid)

	logger := log.WithFields(log.Fields{
		"statusCode": statusCode,
		"deviceUuid": deviceUuid,
	})

	if statusCode == http.StatusOK {
		logger.WithFields(log.Fields{
			"imageID":  image.ID,
			"imageURL": image.Url,
		}).Info("Got image to install")
		//rest.Respond(w, http.StatusOK, image.Url)
		rest.Respond(w, http.StatusOK, "https://blobstore.fi-ts.io/metal/images/os/ubuntu/18.04/img.tar.gz")
	} else {
		logger.Error("No installation image found")
		rest.Respond(w, http.StatusNotFound, nil)
	}
}
