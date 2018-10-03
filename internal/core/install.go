package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func installEndpoint(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["deviceUuid"]

	log.WithField("deviceUuid", uuid).
		Info("Request metal API for an image to install")

	sc, img := srv.GetMetalAPIClient().InstallImage(uuid)

	l := log.WithFields(log.Fields{
		"statusCode": sc,
		"deviceUuid": uuid,
	})

	if sc == http.StatusOK {
		l.WithFields(log.Fields{
			"imageID":  img.ID,
			"imageURL": img.Url,
		}).Info("Got image to install")
		//rest.Respond(w, http.StatusOK, image.Url)
		rest.Respond(w, http.StatusOK, "https://blobstore.fi-ts.io/metal/images/os/ubuntu/18.04/img.tar.gz")
	} else {
		l.Error("No installation image found")
		rest.Respond(w, http.StatusNotFound, nil)
	}
}
