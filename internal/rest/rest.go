package rest

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func respondError(w http.ResponseWriter, errorCode int, msg string) {
	log.WithField("statusCode", errorCode).
		Error(msg)
	respond(w, errorCode, map[string]string{"error": msg})
}

func respond(w http.ResponseWriter, returnCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(returnCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Fatal(err)
	} else {
		log.WithField("statusCode", returnCode).
			Debug(payload)
	}
}
