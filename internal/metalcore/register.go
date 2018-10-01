package metalcore

import (
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func registerDeviceEndpoint(w http.ResponseWriter, r *http.Request) {
	if lshw, err := ioutil.ReadAll(r.Body); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Unable to read body")
	} else {
		log.Info("Register device at Metal API")

		statusCode, device := ApiServer.GetMetalAPIClient().RegisterDevice(string(lshw))

		log.WithFields(log.Fields{
			"statusCode": statusCode,
			"device":     device,
		}).Info("Device registered")

		rest.Respond(w, statusCode, device)
	}
}
