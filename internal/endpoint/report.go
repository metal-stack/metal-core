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
		rest.Respond(h.log, response, http.StatusInternalServerError, nil)
		return
	}

	machineID := request.PathParameter("id")

	h.log.Debug("got report for machine",
		zap.String("machineID", machineID),
		zap.Any("report", report),
	)

	if !report.Success {
		rest.Respond(h.log, response, http.StatusNotAcceptable, nil)
		return
	}

	if h.changeBootOrder {
		ipmiCfg, err := h.apiClient.IPMIConfig(machineID)
		if err != nil {
			rest.Respond(h.log, response, http.StatusInternalServerError, nil)
			return
		}

		err = ipmi.SetBootDisk(h.log, ipmiCfg)
		if err != nil {
			h.log.Error("unable to set boot order of machine to HD",
				zap.String("machineID", machineID),
				zap.Error(err),
			)
			rest.Respond(h.log, response, http.StatusInternalServerError, nil)
			return
		}
	}

	_, err = h.apiClient.FinalizeAllocation(machineID, report.ConsolePassword, report)
	if err != nil {
		h.log.Error("unable to report machine back to api.",
			zap.String("machineID", machineID),
			zap.Error(err),
		)
		rest.Respond(h.log, response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(h.log, response, http.StatusOK, nil)
}
