package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/device"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"

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

	ipmiConn, err := e.ApiClient().IPMIData(devId)
	if err != nil {
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	err = ipmi.SetBootDevHd(ipmiConn)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of device to HD",
			zap.String("deviceID", devId),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	body := &models.ServiceAllocationReport{
		Success:         &report.Success,
		ConsolePassword: &report.ConsolePassword,
		Errormessage:    report.Message,
	}
	params := device.NewAllocationReportParams()
	params.ID = devId
	params.Body = body
	_, err = e.DeviceClient.AllocationReport(params)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to report device back to api.",
			zap.String("deviceID", devId),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(response, http.StatusOK, nil)
}
