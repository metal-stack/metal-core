package core

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/api"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metal-core/internal/netswitch"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

var srv Service

type (
	Service interface {
		GetConfig() domain.Config
		GetMetalAPIClient() api.Client
		GetNetSwitchClient() netswitch.Client
		RunServer()
	}
	service struct {
		api api.Client
		ns  netswitch.Client
	}
)

func NewService(cfg domain.Config) Service {
	srv = service{
		api: api.NewClient(cfg),
		ns:  netswitch.NewClient(cfg),
	}
	return srv
}

func (s service) GetConfig() domain.Config {
	return s.GetMetalAPIClient().GetConfig()
}

func (s service) GetMetalAPIClient() api.Client {
	return s.api
}

func (s service) GetNetSwitchClient() netswitch.Client {
	return s.ns
}

func (s service) RunServer() {
	addr := s.GetConfig().Address
	p := s.GetConfig().Port

	r := mux.NewRouter()
	r.HandleFunc("/v1/boot/{mac}", bootEndpoint).Methods("GET").Name("boot")
	r.HandleFunc("/device/register/{deviceUuid}", registerEndpoint).Methods("POST").Name("register")
	r.HandleFunc("/device/install/{deviceUuid}", installEndpoint).Methods("GET").Name("install")
	r.HandleFunc("/device/report/{deviceUuid}", reportEndpoint).Methods("POST").Name("report")
	r.HandleFunc("/device/ready/{deviceUuid}", readyEndpoint).Methods("POST").Name("ready")
	r.Use(loggingMiddleware)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%v:%d", addr, p),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	log.WithFields(log.Fields{
		"address": addr,
		"port":    p,
	}).Info("Starting API Server")

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		b, _ := ioutil.ReadAll(r.Body)
		h := "{"
		for k, v := range r.Header {
			if len(v) == 1 {
				h += fmt.Sprintf("%v=%v, ", k, v[0])
			} else if len(v) > 1 {
				h += fmt.Sprintf("%v=%v, ", k, v)
			}
		}
		if len(h) > 1 {
			h = h[:len(h)-1]
		}
		h += "}"
		log.WithFields(log.Fields{
			"remoteAddress": r.RemoteAddr,
			"method":        r.Method,
			"protocol":      r.Proto,
			"host":          r.Host,
			"URI":           r.RequestURI,
			"contentLength": r.ContentLength,
			"body":          string(b),
			"headers":       h,
		}).Debug("Got request")
		next.ServeHTTP(w, r)
	})
}
