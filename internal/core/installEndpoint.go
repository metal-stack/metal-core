package core

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	restful "github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func installEndpoint(request *restful.Request, response *restful.Response) {
	devId := request.PathParameter("id")

	zapup.MustRootLogger().Info("Request Metal-API for an image to install",
		zap.String("deviceID", devId),
	)

	sc, devWithToken := srv.API().InstallImage(devId)

	if sc == http.StatusOK && devWithToken != nil && devWithToken.Device != nil {
		zapup.MustRootLogger().Info("Got image to install",
			zap.Int("statusCode", sc),
			zap.Any("deviceWithToken", devWithToken),
		)
		rest.Respond(response, http.StatusOK, devWithToken)
	} else {
		errMsg := "No installation image found"
		zapup.MustRootLogger().Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("deviceID", devId),
		)
		rest.RespondError(response, http.StatusNotFound, errMsg)
	}
}
