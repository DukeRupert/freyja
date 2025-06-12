// internal/metrics/events.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Event publishing metrics
	EventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_published_total",
			Help: "Total number of events published",
		},
		[]string{"event_type", "status"}, // status: success, error
	)

	EventPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "event_publish_duration_seconds",
			Help:    "Duration of event publishing operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"event_type"},
	)

	// Event processing metrics
	EventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total number of events processed by subscribers",
		},
		[]string{"event_type", "subscriber", "status"}, // status: success, error, retry
	)

	EventProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "event_processing_duration_seconds",
			Help:    "Duration of event processing by subscribers",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"event_type", "subscriber"},
	)

	// Business-specific metrics
	CustomerStripeCreations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "customer_stripe_creations_total",
			Help: "Total number of Stripe customer IDs created",
		},
		[]string{"trigger"}, // created, updated
	)

	CustomerStripeUpdates = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "customer_stripe_updates_total",
			Help: "Total number of Stripe customer updates",
		},
		[]string{"status"}, // success, error
	)

	// NATS JetStream metrics
	EventQueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "event_queue_depth",
			Help: "Current depth of event queues",
		},
		[]string{"stream", "consumer"},
	)
)