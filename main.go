package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

func main() {
	runServer()
}

func runServer() {
	router := mux.NewRouter()
	router.HandleFunc("/v1/boot/{mac}", imageEndpoint).Methods("GET").Name("boot")

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

func imageEndpoint(w http.ResponseWriter, r *http.Request) {
	mac := mux.Vars(r)["mac"]
	log.Printf("Serving boot config for mac: %s", mac)
	resp := struct {
		K string   `json:"kernel"`
		I []string `json:"initrd"`
	}{
		K: "http://tinycorelinux.net/7.x/x86/release/distribution_files/vmlinuz64",
		I: []string{
			"http://tinycorelinux.net/7.x/x86/release/distribution_files/rootfs.gz",
			"http://tinycorelinux.net/7.x/x86/release/distribution_files/modules64.gz",
		},
	}

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		panic(err)
	}
}

