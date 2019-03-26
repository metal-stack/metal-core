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

	switch sc {
	case http.StatusOK:
		zapup.MustRootLogger().Info("Got image to install",
			zap.Int("statusCode", sc),
			zap.Any("machineWithToken", machineWithToken),
		)
		rest.Respond(response, http.StatusOK, machineWithToken)
	case http.StatusNotModified:
		zapup.MustRootLogger().Info("Not allocated yet",
			zap.Int("statusCode", sc),
			zap.String("machineID", machineID),
		)
		rest.Respond(response, http.StatusNotModified, nil)
	case http.StatusExpectationFailed:
		zapup.MustRootLogger().Error("Incomplete machine response",
			zap.Int("statusCode", sc),
			zap.String("machineID", machineID),
		)
		rest.Respond(response, http.StatusExpectationFailed, nil)
	default:
		errMsg := "No installation image found"
		zapup.MustRootLogger().Error(errMsg,
			zap.Int("statusCode", sc),
			zap.String("machineID", machineID),
		)
		rest.RespondError(response, http.StatusNotFound, errMsg)
	}
}
