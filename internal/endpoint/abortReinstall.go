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

func (h *endpointHandler) AbortReinstall(request *restful.Request, response *restful.Response) {
	var err error

	cbo := &domain.ChangeBootOrder{}
	err = request.ReadEntity(cbo)
	if err != nil {
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	if !cbo.HD && !cbo.PXE && !cbo.BIOS {
		rest.Respond(response, http.StatusOK, nil)
	}

	machineID := request.PathParameter("id")

	zapup.MustRootLogger().Debug("Got reboot request for machine",
		zap.String("machineID", machineID),
		zap.Any("change boot order", cbo),
	)

	ipmiCfg, err := h.APIClient().IPMIConfig(machineID)
	if err != nil {
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	var boot string
	if cbo.HD {
		boot = "HD"
		err = ipmi.SetBootDisk(ipmiCfg, h.DevMode)
	} else if cbo.PXE {
		boot = "PXE"
		err = ipmi.SetBootPXE(ipmiCfg)
	} else {
		boot = "BIOS"
		err = ipmi.SetBootBios(ipmiCfg, h.DevMode)
	}
	if err != nil {
		zapup.MustRootLogger().Error("Unable to change boot order of machine",
			zap.String("machineID", machineID),
			zap.String("boot", boot),
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(response, http.StatusOK, nil)
}