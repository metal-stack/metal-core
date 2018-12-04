package server

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"
)

func (s server) Run() {
	Init(s.EndpointHandler(s.AppContext))

	// enable CORS for the UI to work
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      restful.DefaultContainer,
	}
	restful.DefaultContainer.Filter(cors.Filter)

	addr := fmt.Sprintf("%v:%d", s.Config.BindAddress, s.Config.Port)

	zapup.MustRootLogger().Info("Starting metal-core",
		zap.String("address", addr),
	)

	http.ListenAndServe(addr, nil)
}
