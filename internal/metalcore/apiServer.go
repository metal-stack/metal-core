package metalcore

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/maas/metalcore/internal/domain"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type MetalcoreApiServer struct {
	metalApiClient domain.MetalAPIClient
}

var ApiServer domain.MetalcoreAPIServer

func NewMetalcoreAPIServer(metalApiClient domain.MetalAPIClient) domain.MetalcoreAPIServer {
	return MetalcoreApiServer{
		metalApiClient: metalApiClient,
	}
}

func (s MetalcoreApiServer) GetMetalAPIClient() domain.MetalAPIClient {
	return s.metalApiClient
}

func (s MetalcoreApiServer) GetConfig() domain.Config {
	return s.GetMetalAPIClient().GetConfig()
}

func (s MetalcoreApiServer) Run() {
	address := s.GetConfig().ServerAddress
	port := s.GetConfig().ServerPort

	router := mux.NewRouter()
	router.HandleFunc("/v1/boot/{mac}", bootEndpoint).Methods("GET").Name("boot")
	router.HandleFunc("/device/register", registerDeviceEndpoint).Methods("POST").Name("register")
	router.HandleFunc("/report/{deviceUuid}", reportDeviceStateEndpoint).Methods("POST").Name("report")
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
			headers += fmt.Sprintf("%v=%v, ", k, v)
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
			"body":          body,
			"headers":       headers,
		}).Debug("Got request")
		next.ServeHTTP(w, r)
	})
}
