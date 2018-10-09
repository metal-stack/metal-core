package core

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/ipmi"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	"github.com/jackpal/gateway"
	log "github.com/sirupsen/logrus"
)

// Report is send back to metal-core after installation finished
type Report struct {
	Success bool   `json:"success" description:"true if installation succeeded"`
	Message string `json:"message" description:"if installation failed, the error message"`
}

func reportEndpoint(w http.ResponseWriter, r *http.Request) {
	reportJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"err": err,
		})).Error("Failed to read request body")
		rest.Respond(w, http.StatusInternalServerError, nil)
		return
	}
	devID := mux.Vars(r)["deviceId"]

	log.Infof("Body: %v", string(reportJSON))

	var report *Report
	err = json.Unmarshal(reportJSON, report)
	if err != nil {
		rest.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	log.WithFields(log.Fields{
		"deviceID": devID,
		"report":   report,
	}).Info("got report for device")

	if !report.Success {
		rest.Respond(w, http.StatusNotAcceptable, nil)
		return
	}

	gateway, err := gateway.DiscoverGateway()
	if err != nil {
		log.Error("Unable to determine gateway for reaching out to ipmi client: ", err)
		rest.Respond(w, http.StatusInternalServerError, nil)
		return
	}
	connection := ipmi.IpmiConnection{
		Hostname:  gateway.String(),
		Interface: "lanplus",
		Port:      6230,
		Username:  "vagrant",
		Password:  "vagrant",
	}
	err = ipmi.SetBootDevHd(connection)
	if err != nil {
		log.Error("Unable to set boot order to hard disk of ipmi client: ", err)
		rest.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	// sc := srv.GetMetalAPIClient().ReportDeviceState(devId, string(state))

	// logger := log.WithFields(log.Fields{
	//	"deviceID":   devId,
	//	"statusCode": sc,
	// })

	// if sc != http.StatusOK {
	// 	logging.Decorate(logger).
	// 		Error("Failed to report device state")
	// } else {
	// 	logger.Info("Device state reported")

	// var sp []domain.SwitchPort
	// sc, sp = srv.GetMetalAPIClient().GetSwitchPorts(devId)

	// logger = log.WithFields(log.Fields{
	// 	"deviceID":    devId,
	// 	"statusCode":  sc,
	// 	"switchPorts": sp,
	// })

	// if sc != http.StatusOK {
	// 	logging.Decorate(logger).
	// 		Error("Failed to retrieve switch ports")
	// } else {
	// 	logger.Info("Retrieved switch ports")

	// 	sc = srv.GetNetSwitchClient().ConfigurePorts(sp)

	// 	logger = log.WithFields(log.Fields{
	// 		"deviceID":    devId,
	// 		"statusCode":  sc,
	// 		"switchPorts": sp,
	// 	})

	// 	if sc != http.StatusOK {
	// 		logging.Decorate(logger).
	// 			Error("Failed to configure switch ports")
	// 	} else {
	// 		logger.Info("Switch ports configured")
	// 	}
	// }

	rest.Respond(w, http.StatusOK, nil)
}
