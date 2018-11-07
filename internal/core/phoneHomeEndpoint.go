package core

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/models"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	restful "github.com/emicklei/go-restful"
	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
)

func phoneHomeEndpoint(request *restful.Request, response *restful.Response) {
	req := &domain.MetalHammerPhoneHomeRequest{}
	if err := request.ReadEntity(req); err != nil {
		errMsg := "Unable to read body"
		zapup.MustRootLogger().Error("Cannot read request",
			zap.Error(err),
		)
		rest.RespondError(response, http.StatusBadRequest, errMsg)
	} else {
		zapup.MustRootLogger().Info("Pass Phone-Home-Request from device to Metal-API")
		params := device.NewPhoneHomeParams()
		params.Body = &models.ServicePhoneHomeRequest{PhoneHomeToken: &(req.PhoneHomeToken)}
		ok, err := srv.API().Device().PhoneHome(params)
		if err != nil {
			rest.RespondError(response, http.StatusBadRequest, err.Error())
		} else if ok == nil {
			rest.RespondError(response, http.StatusBadRequest, "")
		}
		rest.Respond(response, http.StatusOK, ok)
	}
}
