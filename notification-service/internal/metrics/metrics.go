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

	SSEConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "notification_sse_connections_active",
		Help: "Current number of active SSE connections.",
	})

	DeliveryLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "notification_delivery_latency_seconds",
		Help:    "E2E latency from booking event occurrence to SSE broadcast.",
		Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0},
	})

	KafkaConsumedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "notification_kafka_messages_consumed_total",
		Help: "Kafka messages successfully processed.",
	})

	KafkaConsumerErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "notification_kafka_consumer_errors_total",
		Help: "Kafka consumer processing errors.",
	})
)
