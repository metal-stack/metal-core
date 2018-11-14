package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"
)

func (e endpoint) Report(request *restful.Request, response *restful.Response) {
	var err error
	report := &domain.Report{}

	err = request.ReadEntity(report)
	if err != nil {
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

	err = ipmi.SetBootDevHd(e.IpmiConnection)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of device to HD",
			zap.String("deviceID", devId),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(response, http.StatusOK, nil)
}
