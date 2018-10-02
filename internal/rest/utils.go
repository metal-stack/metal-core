package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type ParamsType int

const (
	QueryParameters ParamsType = iota
	PathParameters
)

type Params struct {
	Type ParamsType
	Map  map[string]string
}

func RespondError(w http.ResponseWriter, errorCode int, msg string) {
	log.WithField("statusCode", errorCode).
		Error(msg)
	Respond(w, errorCode, map[string]string{"error": msg})
}

func Respond(w http.ResponseWriter, returnCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(returnCode)
	if payload == nil {
		log.WithFields(log.Fields{
			"statusCode": returnCode,
		}).Info("Sent response")
	} else if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Error(err)
	} else {
		log.WithFields(log.Fields{
			"statusCode": returnCode,
			"payload":    payload,
		}).Info("Sent response")
	}
}

func CreateParams(paramsType ParamsType, keyValuePairs ...string) *Params {
	m := make(map[string]string)
	for i := range keyValuePairs {
		if i%2 == 0 {
			m[keyValuePairs[i]] = keyValuePairs[i+1]
		}
	}
	return &Params{
		Type: paramsType,
		Map:  m,
	}
}

func Get(protocol string, address string, port int, path string, params *Params, domainObject interface{}) int {
	uri := createUri(protocol, address, port, path)
	log.WithFields(log.Fields{
		"method": "GET",
		"URI":    uri,
		"header": "Accept=application/json",
	}).Debug("Rest call")

	request := resty.R().
		SetHeader("Accept", "application/json")
	if params != nil {
		if params.Type == QueryParameters {
			request.SetQueryParams(params.Map)
		} else {
			request.SetPathParams(params.Map)
		}
	}

	response, err := request.Get(uri)
	if err != nil {
		log.Error(err)
	} else {
		unmarshalResponse(response, domainObject)
	}
	return response.StatusCode()
}

func Post(protocol string, address string, port int, path string, params *Params, body interface{}, domainObject interface{}) int {
	response, err := post(protocol, address, port, path, params, body)
	if err != nil {
		log.Error(err)
	} else {
		unmarshalResponse(response, domainObject)
	}
	return response.StatusCode()
}

func PostWithoutResponse(protocol string, address string, port int, path string, params *Params, body interface{}) int {
	response, err := post(protocol, address, port, path, params, body)
	if err != nil {
		log.Error(err)
	}
	return response.StatusCode()
}

func post(protocol string, address string, port int, path string, params *Params, body interface{}) (*resty.Response, error) {
	uri := createUri(protocol, address, port, path)

	logger := log.WithFields(log.Fields{
		"method": "POST",
		"URI":    uri,
		"header": "Content-Type=application/json",
		"body":   body,
	})

	if bodyJson, err := json.Marshal(body); err != nil {
		logger.Error("Failed to marshal body")
		return nil, err
	} else {
		logger.WithField("body", BytesToString(bodyJson)).
			Debug("Rest call")

		request := resty.R().
			SetHeader("Content-Type", "application/json").
			SetBody(bodyJson)
		if params != nil {
			if params.Type == QueryParameters {
				request.SetQueryParams(params.Map)
			} else {
				request.SetPathParams(params.Map)
			}
		}

		if response, err := request.Post(uri); err != nil {
			return nil, err
		} else {
			return response, nil
		}
	}
}

func unmarshalResponse(response *resty.Response, domainObject interface{}) {
	payload := response.Body()
	if err := json.Unmarshal(payload, domainObject); err != nil {
		log.Error(err)
	} else {
		log.WithFields(log.Fields{
			"statusCode": response.StatusCode(),
			"payload":    BytesToString(payload),
		}).Debug("Got response")
	}
}

func createUri(protocol string, address string, port int, path string) string {
	return fmt.Sprintf("%v://%v:%d/%v", sanitizeProtocol(protocol), sanitizeAddress(address), port, sanitizePath(path))
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

func BytesToString(b []byte) string {
	return string(b)
}
