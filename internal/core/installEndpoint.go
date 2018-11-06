package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/maas/metal-core/log"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"
)

func installEndpoint(request *restful.Request, response *restful.Response) {
	devId := request.PathParameter("id")

	log.Get().Info("Request Metal-API for an image to install",
		zap.String("deviceID", devId),
	)

	sc, dev := srv.API().InstallImage(devId)

	if sc == http.StatusOK && dev != nil && dev.Image != nil {
		log.Get().Info("Got image to install",
			zap.Int("statusCode", sc),
			zap.Any("device", dev),
		)
		rest.Respond(response, http.StatusOK, dev)
	} else {
		errMsg := "No installation image found"
		log.Get().Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("deviceID", devId),
		)
		rest.RespondError(response, http.StatusNotFound, errMsg)
	}
}
