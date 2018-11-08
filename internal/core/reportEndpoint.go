package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"
)

// Report is send back to metal-core after installation finished
type Report struct {
	Success bool   `json:"success,omitempty" description:"true if installation succeeded"`
	Message string `json:"message" description:"if installation failed, the error message"`
}

func reportEndpoint(request *restful.Request, response *restful.Response) {
	report := &Report{}
	if err := request.ReadEntity(report); err != nil {
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	devId := request.PathParameter("id")

	zapup.MustRootLogger().Info("Got report for device",
		zap.String("deviceID", devId),
		zap.Any("report", report),
	)

	if !report.Success {
		rest.Respond(response, http.StatusNotAcceptable, nil)
		return
	}

	if err := ipmi.SetBootDevHd(srv.IPMI()); err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of device to HD",
			zap.String("deviceID", devId),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(response, http.StatusOK, nil)
}
