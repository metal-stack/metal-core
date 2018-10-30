package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"github.com/emicklei/go-restful"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	log "github.com/sirupsen/logrus"
)

func registerEndpoint(request *restful.Request, response *restful.Response) {
	req := &domain.MetalHammerRegisterDeviceRequest{}
	if err := request.ReadEntity(req); err != nil {
		errMsg := "Unable to read body"
		logging.Decorate(log.WithFields(log.Fields{
			"err": err,
		})).Error(errMsg)

		rest.RespondError(response, http.StatusBadRequest, errMsg)
	} else {
		devId := request.PathParameter("id")

		log.WithFields(log.Fields{
			"deviceID": devId,
		}).Info("Register device at Metal API")

		sc, dev := srv.GetMetalAPIClient().RegisterDevice(devId, req)

		logger := log.WithFields(log.Fields{
			"deviceID":   devId,
			"statusCode": sc,
			"device":     dev,
		})

		if sc != http.StatusOK {
			errMsg := "Failed to register device"
			logging.Decorate(logger).
				Error(errMsg)
			rest.RespondError(response, http.StatusInternalServerError, errMsg)
		} else {
			logger.Info("Device registered")
			rest.Respond(response, http.StatusOK, dev)
		}
	}
}
