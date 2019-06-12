package core

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core/internal/endpoint"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
)

func (s *coreServer) Run() {
	Init(endpoint.NewHandler(s.AppContext))
	t := time.NewTicker(s.AppContext.Config.ReconfigureSwitchInterval)
	host, _ := os.Hostname()
	go func() {
		for range t.C {
			zapup.MustRootLogger().Info("start periodic switch configuration update")
			err := s.EventHandler().ReconfigureSwitch(host)
			if err != nil {
				zapup.MustRootLogger().Error("unable to fetch and apply switch configuration periodically",
					zap.Error(err))
			}
		}
	}()

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
