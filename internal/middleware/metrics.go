// internal/middleware/metrics.go
package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)
)

// PrometheusMiddleware creates Echo middleware for Prometheus metrics
func PrometheusMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip metrics endpoint to avoid recursion
			if c.Request().URL.Path == "/metrics" {
				return next(c)
			}

			start := time.Now()
			
			// Increment in-flight requests
			HTTPRequestsInFlight.Inc()
			defer HTTPRequestsInFlight.Dec()

			// Process request
			err := next(c)

			// Record metrics
			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(c.Response().Status)
			method := c.Request().Method
			endpoint := c.Path() // Echo route pattern like "/api/v1/customers/:id"

			// If no route pattern, use the actual path
			if endpoint == "" {
				endpoint = c.Request().URL.Path
			}

			// Record total requests
			HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()

			// Record request duration
			HTTPRequestDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration)

			return err
		}
	}
}