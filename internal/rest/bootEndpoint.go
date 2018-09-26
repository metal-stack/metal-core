package rest

import (
	"net/http"
	"github.com/gorilla/mux"
	"log"
	"encoding/json"
)

func bootEndpoint(w http.ResponseWriter, r *http.Request) {
	mac := mux.Vars(r)["mac"]
	log.Printf("Serving boot config for mac: %s", mac)
	resp := struct {
		K string   `json:"kernel"`
		I []string `json:"initrd"`
		C string   `json:"cmdline"`
	}{
		K: "file:///image/pxeboot-kernel",
		I: []string{
			"file:///image/pxeboot-initrd.img",
		},
		C: "console=tty0",
	}

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		panic(err)
	}
}

