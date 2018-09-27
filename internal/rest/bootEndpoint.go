package rest

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func bootEndpoint(w http.ResponseWriter, r *http.Request) {
	mac := mux.Vars(r)["mac"]

	log.Info("Serving boot config for mac \"%v\"", mac)

	response := struct {
		Kernel      string   `json:"kernel"`
		InitRamDisk []string `json:"initrd"`
		CommandLine string   `json:"cmdline"`
	}{
		Kernel: "file:///images/pxeboot-kernel",
		InitRamDisk: []string{
			"file:///images/pxeboot-initrd.img",
		},
		CommandLine: "console=tty0",
	}

	respond(w, http.StatusOK, response)
}
