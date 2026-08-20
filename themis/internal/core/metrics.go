package core

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	themisEvaluationDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "autorix_themis_evaluation_duration_seconds",
			Help:    "Themis ABAC policy evaluation latency in seconds.",
			Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.02, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
	)
	themisEvaluationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_themis_evaluation_total",
			Help: "Total number of Themis policy evaluations partitioned by decision (allow or deny).",
		},
		[]string{"decision"},
	)
)

func init() {
	if err := prometheus.Register(themisEvaluationDuration); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore
		}
	}
	if err := prometheus.Register(themisEvaluationTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore
		}
	}
}
