package endpoint

import (
	"git.f-i-ts.de/cloud-native/metal/metal-core/client/machine"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/rest"
	"git.f-i-ts.de/cloud-native/metal/metal-core/domain"
	"git.f-i-ts.de/cloud-native/metal/metal-core/models"

	"net/http"

	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func (h *endpointHandler) Report(request *restful.Request, response *restful.Response) {
	var err error
	report := &domain.Report{}

	err = request.ReadEntity(report)
	if err != nil {
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	machineID := request.PathParameter("id")

	zapup.MustRootLogger().Info("Got report for machine",
		zap.String("machineID", machineID),
		zap.Any("report", report),
	)

	if !report.Success {
		rest.Respond(response, http.StatusNotAcceptable, nil)
		return
	}

	ipmiConn, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	err = ipmi.SetBootMachineHD(ipmiConn)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of machine to HD",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	body := &models.MetalReportAllocation{
		Success:         &report.Success,
		ConsolePassword: &report.ConsolePassword,
		Errormessage:    report.Message,
	}
	params := machine.NewAllocationReportParams()
	params.ID = machineID
	params.Body = body
	_, err = h.MachineClient.AllocationReport(params)
	if err != nil {
		zapup.MustRootLogger().Error("Unable to report machine back to api.",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(response, http.StatusOK, nil)
}
