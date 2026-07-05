package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests by method, path and status code.",
	}, []string{"method", "path", "code"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1.0},
	}, []string{"method", "path"})

	SlotConflictsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "booking_slot_conflicts_total",
		Help: "Booking attempts rejected due to slot conflict (partial unique index).",
	})

	OutboxPublishedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "booking_outbox_published_total",
		Help: "Outbox records successfully published to Kafka.",
	})

	OutboxErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "booking_outbox_errors_total",
		Help: "Outbox publish errors.",
	})
)
