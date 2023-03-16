package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the collected metrics
type Metrics struct {
	totalErrors *prometheus.CounterVec
}

// New generates new metrics
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

// Init initializes metrics
func (m *Metrics) Init() {
	prometheus.MustRegister(m.totalErrors)
}

// CountError increases error counter for the given operation
func (m *Metrics) CountError(op string) {
	m.totalErrors.With(prometheus.Labels{"operation": op}).Inc()
}
