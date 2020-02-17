package core

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-core/internal/endpoint"
	"github.com/emicklei/go-restful"
	"github.com/metal-stack/metal-lib/zapup"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httppprof "net/http/pprof"

	"go.uber.org/zap"
)

func (s *coreServer) Run() {
	s.initMetrics()

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

func (s *coreServer) initMetrics() {
	logger := zapup.MustRootLogger().Sugar()

	addr := fmt.Sprintf("%v:%d", s.Config.MetricsServerBindAddress, s.Config.MetricsServerPort)

	logger.Infow("starting metrics endpoint", "addr", addr)
	metricsServer := http.NewServeMux()
	metricsServer.Handle("/metrics", promhttp.Handler())
	// see: https://dev.to/davidsbond/golang-debugging-memory-leaks-using-pprof-5di8
	// inspect via
	// go tool pprof -http :8080 localhost:2112/pprof/heap
	// go tool pprof -http :8080 localhost:2112/pprof/goroutine
	metricsServer.Handle("/pprof/heap", httppprof.Handler("heap"))
	metricsServer.Handle("/pprof/goroutine", httppprof.Handler("goroutine"))

	go func() {
		err := http.ListenAndServe(addr, metricsServer)
		if err != nil {
			logger.Errorw("failed to start metrics endpoint, exiting...", "error", err)
			os.Exit(1)
		}
		logger.Errorw("metrics server has stopped unexpectedly without an error")
	}()
}
