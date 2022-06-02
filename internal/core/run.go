package core

import (
	"net/http"
	"os"

	httppprof "net/http/pprof"

	"github.com/emicklei/go-restful/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.uber.org/zap"
)

func (s *server) Run() {
	s.initMetrics()

	Init(s.endpointHandler)

	// enable CORS for the UI to work
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      restful.DefaultContainer,
	}
	restful.DefaultContainer.Filter(cors.Filter)

	s.log.Info("starting metal-core",
		zap.String("address", s.serverAddr),
	)

	s.log.Sugar().Fatal(http.ListenAndServe(s.serverAddr, nil))
}

func (s *server) initMetrics() {
	logger := s.log.Sugar()

	logger.Infow("starting metrics endpoint", "addr", s.metricServerAddr)
	metricsServer := http.NewServeMux()
	metricsServer.Handle("/metrics", promhttp.Handler())
	// see: https://dev.to/davidsbond/golang-debugging-memory-leaks-using-pprof-5di8
	// inspect via
	// go tool pprof -http :8080 localhost:2112/pprof/heap
	// go tool pprof -http :8080 localhost:2112/pprof/goroutine
	metricsServer.Handle("/pprof/heap", httppprof.Handler("heap"))
	metricsServer.Handle("/pprof/goroutine", httppprof.Handler("goroutine"))

	go func() {
		err := http.ListenAndServe(s.metricServerAddr, metricsServer)
		if err != nil {
			logger.Errorw("failed to start metrics endpoint, exiting...", "error", err)
			os.Exit(1)
		}
		logger.Errorw("metrics server has stopped unexpectedly without an error")
	}()
}
