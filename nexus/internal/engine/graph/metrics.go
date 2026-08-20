package graph

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	nexusCheckDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "autorix_nexus_check_duration_seconds",
			Help:    "Nexus ReBAC graph check evaluation latency in seconds.",
			Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.02, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
	)
	nexusCheckTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_nexus_check_total",
			Help: "Total number of Nexus check evaluations partitioned by decision (allow or deny).",
		},
		[]string{"decision"},
	)
)

func init() {
	if err := prometheus.Register(nexusCheckDuration); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore
		}
	}
	if err := prometheus.Register(nexusCheckTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore
		}
	}
}
