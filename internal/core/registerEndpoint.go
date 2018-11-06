package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/log"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
)

func registerEndpoint(request *restful.Request, response *restful.Response) {
	req := &domain.MetalHammerRegisterDeviceRequest{}
	if err := request.ReadEntity(req); err != nil {
		errMsg := "Unable to read body"
		log.Get().Error("Cannot read request",
			zap.Error(err),
		)
		rest.RespondError(response, http.StatusBadRequest, errMsg)
	} else {
		devId := request.PathParameter("id")

		log.Get().Info("Register device at Metal-API",
			zap.String("deviceID", devId),
		)

		sc, dev := srv.API().RegisterDevice(devId, req)

		if sc != http.StatusOK {
			errMsg := "Failed to register device"
			log.Get().Error(errMsg,
				zap.Int("statusCode", sc),
				zap.String("deviceID", devId),
				zap.Any("device", dev),
				zap.Error(err),
			)
			rest.RespondError(response, http.StatusInternalServerError, errMsg)
		} else {
			log.Get().Info("Device registered",
				zap.Int("statusCode", sc),
				zap.Any("device", dev),
			)
			rest.Respond(response, http.StatusOK, dev)
		}
	}
}
