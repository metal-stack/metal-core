package metalcore

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func reportEndpoint(w http.ResponseWriter, r *http.Request) {
	if state, err := ioutil.ReadAll(r.Body); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to read request body")
	} else {
		deviceUuid := mux.Vars(r)["deviceUuid"]

		log.WithFields(log.Fields{
			"deviceUuid": deviceUuid,
			"state":      state,
		}).Info("Inform Metal API about device state")

		statusCode := srv.GetMetalAPIClient().ReportDeviceState(deviceUuid, rest.BytesToString(state))

		logger := log.WithFields(log.Fields{
			"deviceUuid": deviceUuid,
			"statusCode": statusCode,
		})

		if statusCode != http.StatusOK {
			logger.Error("Failed to report device state")
		} else {
			logger.Info("Device state reported")

			statusCode, switchPorts := srv.GetMetalAPIClient().GetSwitchPorts(deviceUuid)

			logger = log.WithFields(log.Fields{
				"deviceUuid":  deviceUuid,
				"statusCode":  statusCode,
				"switchPorts": switchPorts,
			})

			if statusCode != http.StatusOK {
				logger.Error("Failed to retrieve switch ports")
			} else {
				logger.Info("Retrieved switch ports")

				statusCode := srv.GetNetSwitchClient().ConfigurePorts(switchPorts)

				logger = log.WithFields(log.Fields{
					"deviceUuid":  deviceUuid,
					"statusCode":  statusCode,
					"switchPorts": switchPorts,
				})

				if statusCode != http.StatusOK {
					logger.Error("Failed to configure switch ports")
				} else {
					logger.Info("Switch ports configured")
				}
			}
		}

		rest.Respond(w, statusCode, nil)
	}
}
