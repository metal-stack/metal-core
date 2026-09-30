package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	totalErrors *prometheus.CounterVec
}

func New() *Metrics {

	totalErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "metal_core_total_errors",
		Help: "total number of errors",
	},
		[]string{"operation"},
	)

	return &Metrics{
		totalErrors: totalErrors,
	}
}

func (m *Metrics) Init() {
	prometheus.MustRegister(m.totalErrors)
}

func (m *Metrics) CountError(op string) {
	m.totalErrors.With(prometheus.Labels{"operation": op}).Inc()
}
