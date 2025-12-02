package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Prometheus metrics collectors
type Metrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
	responseSize     *prometheus.HistogramVec
}

// NewMetrics creates and registers Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "freyja"
	}

	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),
		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "path", "status"},
		),
	}

	// Register metrics
	prometheus.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		m.responseSize,
	)

	return m
}

// Middleware returns an HTTP middleware that records metrics
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track in-flight requests
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		// Wrap response writer to capture status and size
		wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)
		path := normalizePath(r.URL.Path)

		m.requestsTotal.WithLabelValues(r.Method, path, status).Inc()
		m.requestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
		m.responseSize.WithLabelValues(r.Method, path, status).Observe(float64(wrapped.bytesWritten))
	})
}

// Handler returns the Prometheus metrics HTTP handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// metricsResponseWriter wraps http.ResponseWriter to capture status and size
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// normalizePath normalizes URL paths for metrics labels
// This prevents high cardinality from dynamic path segments like IDs
func normalizePath(path string) string {
	// For common patterns, normalize dynamic segments
	// This is a simple implementation - enhance as needed for your routes

	// Skip static files
	if len(path) >= 8 && path[:8] == "/static/" {
		return "/static/*"
	}

	// Normalize admin paths with IDs
	if len(path) >= 7 && path[:7] == "/admin/" {
		return normalizeAdminPath(path)
	}

	// Normalize product paths
	if len(path) >= 10 && path[:10] == "/products/" {
		if path == "/products/" {
			return "/products/"
		}
		return "/products/:slug"
	}

	// Normalize account subscription paths
	if len(path) >= 23 && path[:23] == "/account/subscriptions/" {
		if len(path) > 23 {
			return "/account/subscriptions/:id"
		}
	}

	// Normalize order confirmation
	if len(path) >= 19 && path[:19] == "/order-confirmation" {
		return "/order-confirmation"
	}

	return path
}

// normalizeAdminPath normalizes admin paths with dynamic segments
func normalizeAdminPath(path string) string {
	// /admin/products/:id
	// /admin/products/:id/edit
	// /admin/products/:product_id/skus/:sku_id
	// /admin/orders/:id
	// /admin/subscriptions/:id

	segments := splitPath(path)
	if len(segments) < 2 {
		return path
	}

	// /admin/{resource}
	if len(segments) == 2 {
		return path
	}

	// /admin/{resource}/{id} or /admin/{resource}/new
	if len(segments) == 3 {
		if segments[2] == "new" {
			return path
		}
		return "/admin/" + segments[1] + "/:id"
	}

	// /admin/{resource}/{id}/edit or /admin/{resource}/{id}/{sub}
	if len(segments) == 4 {
		if segments[3] == "edit" {
			return "/admin/" + segments[1] + "/:id/edit"
		}
		return "/admin/" + segments[1] + "/:id/" + segments[3]
	}

	// /admin/{resource}/{id}/{sub}/{sub_id}
	if len(segments) >= 5 {
		if segments[4] == "new" {
			return "/admin/" + segments[1] + "/:id/" + segments[3] + "/new"
		}
		if len(segments) == 5 {
			return "/admin/" + segments[1] + "/:id/" + segments[3] + "/:sub_id"
		}
		if len(segments) == 6 && segments[5] == "edit" {
			return "/admin/" + segments[1] + "/:id/" + segments[3] + "/:sub_id/edit"
		}
	}

	return path
}

// splitPath splits a path into segments
func splitPath(path string) []string {
	var segments []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				segments = append(segments, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		segments = append(segments, path[start:])
	}
	return segments
}
