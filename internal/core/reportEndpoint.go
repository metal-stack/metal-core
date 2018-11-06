package core

import (
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
)

// Report is send back to metal-core after installation finished
type Report struct {
	Success bool   `json:"success" description:"true if installation succeeded" optional:"true"`
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

	connection := ipmi.IpmiConnection{
		// Requires gateway of the control plane for running in Metal Lab... this is just a quick workaround for the poc
		Hostname:  srv.GetConfig().IP[:strings.LastIndex(srv.GetConfig().IP, ".")] + ".1",
		Interface: "lanplus",
		Port:      6230,
		Username:  "vagrant",
		Password:  "vagrant",
	}
	if err := ipmi.SetBootDevHd(connection); err != nil {
		zapup.MustRootLogger().Error("Unable to set boot order of IPMI client to HD",
			zap.Error(err),
		)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(response, http.StatusOK, nil)
}
