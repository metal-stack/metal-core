package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
)

type (
	Params  map[string]string
	Request struct {
		Protocol string
		Address  string
		Port     int
		Path     string
		Params   *Params
	}
)

func NewRequest(protocol string, address string, port int, path string, params *Params) *Request {
	return &Request{
		Protocol: protocol,
		Address:  address,
		Port:     port,
		Path:     path,
		Params:   params,
	}
}

func (r *Request) Get() *resty.Response {
	uri := r.createUri()
	log.WithFields(log.Fields{
		"method": "GET",
		"URI":    uri,
		"header": "Accept=application/json",
	}).Debug("Rest call")

	req := resty.R().
		SetHeader("Accept", "application/json")
	if r.Params != nil {
		req.SetQueryParams(*r.Params)
	}

	resp, err := req.Get(uri)
	if err != nil {
		log.Error(err)
		return nil
	} else {
		return resp
	}
}

func (r *Request) Post(body interface{}) *resty.Response {
	resp, err := r.post(body)
	if err != nil {
		log.Error(err)
		return nil
	} else {
		return resp
	}
}

func (r *Request) post(body interface{}) (*resty.Response, error) {
	uri := r.createUri()

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
		logger.WithField("body", string(bodyJson)).
			Debug("Rest call")

		req := resty.R().
			SetHeader("Content-Type", "application/json").
			SetBody(bodyJson)
		if r.Params != nil {
			req.SetQueryParams(*r.Params)
		}

		if resp, err := req.Post(uri); err != nil {
			return nil, err
		} else {
			return resp, nil
		}
	}
}

func (r *Request) createUri() string {
	return fmt.Sprintf("%v://%v:%d/%v", sanitizeProtocol(r.Protocol), sanitizeAddress(r.Address), r.Port, sanitizePath(r.Path))
}

func Respond(w http.ResponseWriter, sc int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	if body == nil {
		log.WithFields(log.Fields{
			"statusCode": sc,
		}).Info("Sent response")
	} else if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error(err)
	} else {
		log.WithFields(log.Fields{
			"statusCode": sc,
			"body":       body,
		}).Info("Sent response")
	}
}

func CreateQueryParams(kv ...string) *Params {
	p := make(Params)
	for i := range kv {
		if i%2 == 0 {
			p[kv[i]] = kv[i+1]
		}
	}
	return &p
}

func Unmarshal(resp *resty.Response, v interface{}) {
	body := resp.Body()
	if err := json.Unmarshal(body, v); err != nil {
		log.Error(err)
	} else {
		log.WithFields(log.Fields{
			"statusCode": resp.StatusCode(),
			"body":       string(body),
		}).Debug("Got response")
	}
}

func sanitizeProtocol(proto string) string {
	return strings.TrimSpace(proto)
}

func sanitizeAddress(addr string) string {
	return strings.TrimSpace(addr)
}

func sanitizePath(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "/") {
		return path[1:]
	} else {
		return path
	}
}
