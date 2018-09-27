package rest

import (
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func RunAPIServer(port int) {
	router := mux.NewRouter()
	router.HandleFunc("/v1/boot/{mac}", bootEndpoint).Methods("GET").Name("boot")
	router.Use(loggingMiddleware)

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	log.WithField("port", port).Info("Starting API Server")

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithField("URI", r.RequestURI).
			Debug("Request")
		next.ServeHTTP(w, r)
	})
}
