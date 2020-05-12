package endpoint

import (
	"net/http"

	"github.com/metal-stack/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful"
	"github.com/metal-stack/metal-lib/zapup"
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

	zapup.MustRootLogger().Debug("Got report for machine",
		zap.String("machineID", machineID),
		zap.Any("report", report),
	)

	if !report.Success {
		rest.Respond(response, http.StatusNotAcceptable, nil)
		return
	}

	if h.Config.ChangeBootOrder {
		ipmiCfg, err := h.APIClient().IPMIConfig(machineID, h.Compliance)
		if err != nil {
			rest.Respond(response, http.StatusInternalServerError, nil)
			return
		}

		err = ipmi.SetBootDisk(ipmiCfg)
		if err != nil {
			zapup.MustRootLogger().Error("Unable to set boot order of machine to HD",
				zap.String("machineID", machineID),
				zap.Error(err),
			)
			rest.Respond(response, http.StatusInternalServerError, nil)
			return
		}
	}

	_, err = h.APIClient().FinalizeAllocation(machineID, report.ConsolePassword, report)
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
