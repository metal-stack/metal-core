package core

import (
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/rest"
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
		uuid := mux.Vars(r)["deviceUuid"]

		log.WithFields(log.Fields{
			"deviceUuid": uuid,
			"state":      state,
		}).Info("Inform Metal API about device state")

		sc := srv.GetMetalAPIClient().ReportDeviceState(uuid, string(state))

		l := log.WithFields(log.Fields{
			"deviceUuid": uuid,
			"statusCode": sc,
		})

		if sc != http.StatusOK {
			l.Error("Failed to report device state")
		} else {
			l.Info("Device state reported")

			var sp []domain.SwitchPort
			sc, sp = srv.GetMetalAPIClient().GetSwitchPorts(uuid)

			l = log.WithFields(log.Fields{
				"deviceUuid":  uuid,
				"statusCode":  sc,
				"switchPorts": sp,
			})

			if sc != http.StatusOK {
				l.Error("Failed to retrieve switch ports")
			} else {
				l.Info("Retrieved switch ports")

				sc = srv.GetNetSwitchClient().ConfigurePorts(sp)

				l = log.WithFields(log.Fields{
					"deviceUuid":  uuid,
					"statusCode":  sc,
					"switchPorts": sp,
				})

				if sc != http.StatusOK {
					l.Error("Failed to configure switch ports")
				} else {
					l.Info("Switch ports configured")
				}
			}
		}

		rest.Respond(w, sc, nil)
	}
}
