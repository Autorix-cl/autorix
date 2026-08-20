package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	aegisRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_aegis_requests_total",
			Help: "Total number of proxy requests processed by Aegis partitioned by status code.",
		},
		[]string{"status"},
	)
	pipelineStageDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "autorix_aegis_pipeline_stage_duration_seconds",
			Help:    "Duration in seconds for each stage in the Aegis zero-trust pipeline.",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"stage"},
	)
)

func init() {
	if err := prometheus.Register(aegisRequestsTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore
		}
	}
	if err := prometheus.Register(pipelineStageDuration); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// ignore
		}
	}
}
