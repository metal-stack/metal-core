package endpoint

import (
	"net/http"

	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func (h *endpointHandler) Install(request *restful.Request, response *restful.Response) {
	machineID := request.PathParameter("id")

	zapup.MustRootLogger().Info("Request Metal-API for an image to install",
		zap.String("machineID", machineID),
	)

	sc, machineWithToken := h.APIClient().InstallImage(machineID)

	if sc == http.StatusOK {
		zapup.MustRootLogger().Info("Got image to install",
			zap.Int("statusCode", sc),
			zap.Any("machineWithToken", machineWithToken),
		)
		rest.Respond(response, http.StatusOK, machineWithToken)
		return
	}

	errMsg := "No installation image found"
	zapup.MustRootLogger().Error(errMsg,
		zap.Int("statusCode", sc),
		zap.String("machineID", machineID),
	)
	rest.RespondError(response, http.StatusNotFound, errMsg)
}
