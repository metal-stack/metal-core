package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/logging"

	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/netswitch"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type (
	Service interface {
		GetConfig() *domain.Config
		GetMetalAPIClient() api.Client
		GetNetSwitchClient() netswitch.Client
		GetServer() *http.Server
		RunServer()
	}
	service struct {
		server          *http.Server
		apiClient       api.Client
		netSwitchClient netswitch.Client
	}
)

var srv Service

func NewService(cfg *domain.Config) Service {
	srv = service{
		server:          &http.Server{},
		apiClient:       api.NewClient(cfg),
		netSwitchClient: netswitch.NewClient(cfg),
	}
	return srv
}

func (s service) GetConfig() *domain.Config {
	return s.GetMetalAPIClient().GetConfig()
}

func (s service) GetMetalAPIClient() api.Client {
	return s.apiClient
}

func (s service) GetNetSwitchClient() netswitch.Client {
	return s.netSwitchClient
}

func (s service) GetServer() *http.Server {
	return s.server
}

func (s service) RunServer() {
	addr := s.GetConfig().Address
	port := s.GetConfig().Port

	router := mux.NewRouter()
	router.HandleFunc("/v1/boot/{mac}", bootEndpoint).Methods(http.MethodGet).Name("boot")
	router.HandleFunc("/device/register/{deviceId}", registerEndpoint).Methods(http.MethodPost).Name("register")
	router.HandleFunc("/device/install/{deviceId}", installEndpoint).Methods(http.MethodGet).Name("install")
	router.HandleFunc("/device/report/{deviceId}", reportEndpoint).Methods(http.MethodPost).Name("report")

	router.Use(loggingMiddleware)

	server := s.GetServer()
	server.Addr = fmt.Sprintf("%v:%d", addr, port)
	server.Handler = router

	log.WithFields(log.Fields{
		"address": addr,
		"port":    port,
	}).Info("Starting metal-core")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logging.Decorate(log.WithField("error", err)).
			Fatal("Cannot start Metal-Core server")
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		headers := "{"
		for k, v := range r.Header {
			if len(v) == 1 {
				headers += fmt.Sprintf("%v=%v, ", k, v[0])
			} else if len(v) > 1 {
				headers += fmt.Sprintf("%v=%v, ", k, v)
			}
		}
		if len(headers) > 1 {
			headers = headers[:len(headers)-1]
		}
		headers += "}"
		log.WithFields(log.Fields{
			"remoteAddress": r.RemoteAddr,
			"method":        r.Method,
			"protocol":      r.Proto,
			"host":          r.Host,
			"URI":           r.RequestURI,
			"contentLength": r.ContentLength,
			"body":          string(body),
			"headers":       headers,
		}).Debug("Got request")

		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		next.ServeHTTP(w, r)
	})
}
