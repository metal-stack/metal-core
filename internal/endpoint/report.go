package endpoint

import (
	"net/http"

	"github.com/metal-stack/metal-core/internal/ipmi"
	"github.com/metal-stack/metal-core/internal/rest"
	"github.com/metal-stack/metal-core/pkg/domain"

	"github.com/emicklei/go-restful/v3"
	"go.uber.org/zap"
)

func (h *endpointHandler) Report(request *restful.Request, response *restful.Response) {
	var err error

	report := &domain.Report{}
	err = request.ReadEntity(report)
	if err != nil {
		rest.Respond(h.Log, response, http.StatusInternalServerError, nil)
		return
	}

	machineID := request.PathParameter("id")

	h.Log.Debug("got report for machine",
		zap.String("machineID", machineID),
		zap.Any("report", report),
	)

	if !report.Success {
		rest.Respond(h.Log, response, http.StatusNotAcceptable, nil)
		return
	}

	if h.Config.ChangeBootOrder {
		ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
		if err != nil {
			rest.Respond(h.Log, response, http.StatusInternalServerError, nil)
			return
		}

		err = ipmi.SetBootDisk(h.Log, ipmiCfg)
		if err != nil {
			h.Log.Error("unable to set boot order of machine to HD",
				zap.String("machineID", machineID),
				zap.Error(err),
			)
			rest.Respond(h.Log, response, http.StatusInternalServerError, nil)
			return
		}
	}

	_, err = h.APIClient().FinalizeAllocation(machineID, report.ConsolePassword, report)
	if err != nil {
		h.Log.Error("unable to report machine back to api.",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
		rest.Respond(h.Log, response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(h.Log, response, http.StatusOK, nil)
}
