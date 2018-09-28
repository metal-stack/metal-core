package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

func RespondError(w http.ResponseWriter, errorCode int, msg string) {
	log.WithField("statusCode", errorCode).
		Error(msg)
	Respond(w, errorCode, map[string]string{"error": msg})
}

func Respond(w http.ResponseWriter, returnCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(returnCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(err)
	} else {
		log.WithFields(log.Fields{
			"statusCode": returnCode,
			"payload":    payload,
		}).Info("Sent response")
	}
}

func CreateQueryParameters(keyValuePairs ...string) map[string]string {
	m := make(map[string]string)
	for i := range keyValuePairs {
		if i%2 == 0 {
			m[keyValuePairs[i]] = keyValuePairs[i+1]
		}
	}
	return m
}

func Get(protocol string, address string, port int, path string, queryParameters map[string]string, domainObject interface{}) int {
	response, err := resty.R().
		SetQueryParams(queryParameters).
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%v://%v:%d/%v", sanitizeProtocol(protocol), sanitizeAddress(address), port, sanitizePath(path)))
	payload := response.Body()
	if err != nil {
		log.Error(err)
	} else if err := json.Unmarshal(payload, domainObject); err != nil {
		log.Error(err)
	} else {
		log.WithFields(log.Fields{
			"statusCode": response.StatusCode(),
			"payload":    string(payload),
		}).Info("Got response")
	}
	return response.StatusCode()
}

func sanitizeProtocol(protocol string) string {
	return strings.TrimSpace(protocol)
}

func sanitizeAddress(address string) string {
	return strings.TrimSpace(address)
}

func sanitizePath(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "/") {
		return path[1:]
	} else {
		return path
	}
}
