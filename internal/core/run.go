package core

import (
	"fmt"
	"net/http"

	httppprof "net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *coreServer) Run() {
	logger := s.Log.Sugar()

	metricsAddr := fmt.Sprintf("%v:%d", s.Config.MetricsServerBindAddress, s.Config.MetricsServerPort)

	logger.Infow("starting metrics endpoint", "addr", metricsAddr)
	metricsServer := http.NewServeMux()
	metricsServer.Handle("/metrics", promhttp.Handler())
	// see: https://dev.to/davidsbond/golang-debugging-memory-leaks-using-pprof-5di8
	// inspect via
	// go tool pprof -http :8080 localhost:2112/pprof/heap
	// go tool pprof -http :8080 localhost:2112/pprof/goroutine
	metricsServer.Handle("/pprof/heap", httppprof.Handler("heap"))
	metricsServer.Handle("/pprof/goroutine", httppprof.Handler("goroutine"))

	s.Log.Sugar().Fatal(http.ListenAndServe(metricsAddr, metricsServer))
}
