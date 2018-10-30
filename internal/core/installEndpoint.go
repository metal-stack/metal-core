package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/emicklei/go-restful"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func installEndpoint(request *restful.Request, response *restful.Response) {
	devId := request.PathParameter("id")

	log.WithField("deviceID", devId).
		Info("Request metal API for an image to install")

	sc, dev := srv.GetMetalAPIClient().InstallImage(devId)

	logger := log.WithFields(log.Fields{
		"statusCode": sc,
		"deviceID":   devId,
		"dev":        dev,
	})

	if sc == http.StatusOK && dev != nil && dev.Image != nil {
		logger.WithFields(log.Fields{
			"imageID":  dev.Image.ID,
			"imageURL": dev.Image.URL,
		}).Info("Got image to install")
		rest.Respond(response, http.StatusOK, dev)
	} else {
		errMsg := "No installation image found"
		logging.Decorate(logger).
			Error(errMsg)
		rest.RespondError(response, http.StatusNotFound, errMsg)
	}
}
