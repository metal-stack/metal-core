package core

import (
	"github.com/emicklei/go-restful"
	"net/http"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	log "github.com/sirupsen/logrus"
)

// Report is send back to metal-core after installation finished
type Report struct {
	Success bool   `json:"success" description:"true if installation succeeded"`
	Message string `json:"message" description:"if installation failed, the error message"`
}

func reportEndpoint(request *restful.Request, response *restful.Response) {
	report := &Report{}
	if err := request.ReadEntity(report); err != nil {
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	devId := request.PathParameter("id")

	log.WithFields(log.Fields{
		"deviceID": devId,
		"report":   report,
	}).Info("got report for device")

	if !report.Success {
		rest.Respond(response, http.StatusNotAcceptable, nil)
		return
	}

	connection := ipmi.IpmiConnection{
		// Requires gateway of the control plane for running in Metal Lab... this is just a quick workaround for the poc
		Hostname:  srv.GetConfig().ControlPlaneIP[:strings.LastIndex(srv.GetConfig().ControlPlaneIP, ".")] + ".1",
		Interface: "lanplus",
		Port:      6230,
		Username:  "vagrant",
		Password:  "vagrant",
	}
	if err := ipmi.SetBootDevHd(connection); err != nil {
		log.Error("Unable to set boot order to hard disk of ipmi client: ", err)
		rest.Respond(response, http.StatusInternalServerError, nil)
		return
	}

	rest.Respond(response, http.StatusOK, nil)
}
