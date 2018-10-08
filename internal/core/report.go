package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func reportEndpoint(w http.ResponseWriter, r *http.Request) {
	if state, err := ioutil.ReadAll(r.Body); err != nil {
		logging.Decorate(log.WithFields(log.Fields{
			"err": err,
		})).Error("Failed to read request body")
	} else {
		devId := mux.Vars(r)["deviceId"]

		log.WithFields(log.Fields{
			"deviceID": devId,
			"state":    state,
		}).Info("Inform Metal API about device state")

		sc := srv.GetMetalAPIClient().ReportDeviceState(devId, string(state))

		logger := log.WithFields(log.Fields{
			"deviceID":   devId,
			"statusCode": sc,
		})

		if sc != http.StatusOK {
			logging.Decorate(logger).
				Error("Failed to report device state")
		} else {
			logger.Info("Device state reported")

			var sp []domain.SwitchPort
			sc, sp = srv.GetMetalAPIClient().GetSwitchPorts(devId)

			logger = log.WithFields(log.Fields{
				"deviceID":    devId,
				"statusCode":  sc,
				"switchPorts": sp,
			})

			if sc != http.StatusOK {
				logging.Decorate(logger).
					Error("Failed to retrieve switch ports")
			} else {
				logger.Info("Retrieved switch ports")

				sc = srv.GetNetSwitchClient().ConfigurePorts(sp)

				logger = log.WithFields(log.Fields{
					"deviceID":    devId,
					"statusCode":  sc,
					"switchPorts": sp,
				})

				if sc != http.StatusOK {
					logging.Decorate(logger).
						Error("Failed to configure switch ports")
				} else {
					logger.Info("Switch ports configured")
				}
			}
		}

		rest.Respond(w, sc, nil)
	}
}
