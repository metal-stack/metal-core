package core

import (
	"fmt"
	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	"net/http"
)

func (s *coreServer) Run() {
	Init(endpoint.NewHandler(s.AppContext))

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

	zapup.MustRootLogger().Sugar().Fatal(http.ListenAndServe(addr, nil))
}
