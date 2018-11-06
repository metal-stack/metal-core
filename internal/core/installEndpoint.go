package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"
)

func installEndpoint(request *restful.Request, response *restful.Response) {
	devId := request.PathParameter("id")

	zapup.MustRootLogger().Info("Request Metal-API for an image to install",
		zap.String("deviceID", devId),
	)

	sc, dev := srv.GetMetalAPIClient().InstallImage(devId)

	if sc == http.StatusOK && dev != nil && dev.Image != nil {
		zapup.MustRootLogger().Info("Got image to install",
			zap.Int("statusCode", sc),
			zap.Any("device", dev),
		)
		rest.Respond(response, http.StatusOK, dev)
	} else {
		errMsg := "No installation image found"
		zapup.MustRootLogger().Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("deviceID", devId),
		)
		rest.RespondError(response, http.StatusNotFound, errMsg)
	}
}
