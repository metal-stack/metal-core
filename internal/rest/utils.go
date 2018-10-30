package rest

import (
	"errors"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	"github.com/emicklei/go-restful"
	log "github.com/sirupsen/logrus"
)

func RespondError(response *restful.Response, statusCode int, errMsg string) {
	if err := response.WriteError(statusCode, errors.New(errMsg)); err == nil {
		response.Flush()
		log.WithFields(log.Fields{
			"statusCode": statusCode,
			"error":      errMsg,
		}).Error("Sent error response")
	} else {
		logging.Decorate(log.WithFields(log.Fields{})).
			Error(err)
	}
}

func Respond(response *restful.Response, statusCode int, body interface{}) {
	if body == nil {
		log.WithField("statusCode", statusCode).
			Info("Sent response")
	} else if err := response.WriteEntity(body); err != nil {
		logging.Decorate(log.WithFields(log.Fields{})).
			Error(err)
	} else {
		response.Flush()
		log.WithFields(log.Fields{
			"statusCode": statusCode,
			"body":       body,
		}).Info("Sent response")
	}
}
