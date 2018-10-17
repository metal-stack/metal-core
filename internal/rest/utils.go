package rest

import (
	"encoding/json"
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func RespondError(w http.ResponseWriter, code int, errMsg string) {
	Respond(w, code, fmt.Sprintf("Error: %v", errMsg))
}

func Respond(w http.ResponseWriter, sc int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	if body == nil {
		log.WithField("statusCode", sc).
			Info("Sent response")
	} else if err := json.NewEncoder(w).Encode(body); err != nil {
		logging.Decorate(log.WithFields(log.Fields{})).
			Error(err)
	} else {
		log.WithFields(log.Fields{
			"statusCode": sc,
			"body":       body,
		}).Info("Sent response")
	}
}
