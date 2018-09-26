package rest

import (
	"github.com/gorilla/mux"
	"net/http"
	"time"
	"log"
)

func RunAPIServer() {
	router := mux.NewRouter()
	router.HandleFunc("/v1/boot/{mac}", bootEndpoint).Methods("GET").Name("boot")

	srv := &http.Server{
		Addr: "0.0.0.0:4242",
		// Good practice to set timeouts to avoid Slowloris attacks
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
