package metalcore

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/metalapi"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/netswitch"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/rest"
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
		GetMetalAPIClient() metalapi.Client
		GetNetSwitchClient() netswitch.Client
		RunServer()
	}
	service struct {
		metalApiClient  metalapi.Client
		netSwitchClient netswitch.Client
	}
)

func NewService(config domain.Config) Service {
	srv = service{
		metalApiClient:  metalapi.NewClient(config),
		netSwitchClient: netswitch.NewClient(config),
	}
	return srv
}

func RunServer() {
	srv.RunServer()
}

func (s service) GetConfig() domain.Config {
	return s.GetMetalAPIClient().GetConfig()
}

func (s service) GetMetalAPIClient() metalapi.Client {
	return s.metalApiClient
}

func (s service) GetNetSwitchClient() netswitch.Client {
	return s.netSwitchClient
}

func (s service) RunServer() {
	address := s.GetConfig().ServerAddress
	port := s.GetConfig().ServerPort

	router := mux.NewRouter()
	router.HandleFunc("/v1/boot/{mac}", bootEndpoint).Methods("GET").Name("boot")
	router.HandleFunc("/device/register/{deviceUuid}", registerEndpoint).Methods("POST").Name("register")
	router.HandleFunc("/device/install/{deviceUuid}", installEndpoint).Methods("GET").Name("install")
	router.HandleFunc("/device/report/{deviceUuid}", reportEndpoint).Methods("POST").Name("report")
	router.HandleFunc("/device/ready/{deviceUuid}", readyEndpoint).Methods("POST").Name("ready")
	router.Use(loggingMiddleware)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%v:%d", address, port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	log.WithFields(log.Fields{
		"address": address,
		"port":    port,
	}).Info("Starting API Server")

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
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
			"body":          rest.BytesToString(body),
			"headers":       headers,
		}).Debug("Got request")
		next.ServeHTTP(w, r)
	})
}
