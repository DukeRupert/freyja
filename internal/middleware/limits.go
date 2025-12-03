package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// MaxBodySize limits the size of request bodies.
// If no size is provided, DefaultMaxBodySize (10MB) is used.
// If the request body exceeds maxBytes, it returns 413 Request Entity Too Large.
func MaxBodySize(maxBytes ...int64) func(http.Handler) http.Handler {
	var limit int64
	if len(maxBytes) > 0 {
		limit = maxBytes[0]
	} else {
		limit = DefaultMaxBodySize
	}

	return maxBodySizeWithLimit(limit)
}

func maxBodySizeWithLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only limit if there's a body
			if r.Body != nil && r.ContentLength > maxBytes {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			// Wrap the body with a limited reader
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// Common size limits
const (
	KB = 1024
	MB = 1024 * KB

	// DefaultMaxBodySize is the default maximum request body size (10MB)
	DefaultMaxBodySize = 10 * MB

	// SmallMaxBodySize is for endpoints that don't need large bodies (1MB)
	SmallMaxBodySize = 1 * MB

	// LargeMaxBodySize is for file uploads (50MB)
	LargeMaxBodySize = 50 * MB
)

// Timeout adds a timeout to request processing.
// If no duration is provided, DefaultTimeout (30s) is used.
// If the handler doesn't complete within the timeout, it returns 503 Service Unavailable.
func Timeout(timeout ...time.Duration) func(http.Handler) http.Handler {
	var duration time.Duration
	if len(timeout) > 0 {
		duration = timeout[0]
	} else {
		duration = DefaultTimeout
	}

	return timeoutWithDuration(duration)
}

func timeoutWithDuration(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal completion
			done := make(chan struct{})

			// Wrap the response writer to detect if we've started writing
			tw := &timeoutWriter{
				ResponseWriter: w,
				done:           done,
			}

			// Run the handler in a goroutine
			go func() {
				defer close(done)
				next.ServeHTTP(tw, r.WithContext(ctx))
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				// Handler completed normally
				return
			case <-ctx.Done():
				// Timeout occurred
				tw.mu.Lock()
				defer tw.mu.Unlock()

				if !tw.wroteHeader {
					// Only send error if we haven't started responding
					w.WriteHeader(http.StatusServiceUnavailable)
					w.Write([]byte("Request timeout"))
				}
				// If we've already started writing, we can't do much
				// The client will receive a truncated response
			}
		})
	}
}

// Common timeout values
const (
	// DefaultTimeout is the default request timeout (30 seconds)
	DefaultTimeout = 30 * time.Second

	// ShortTimeout is for quick operations (5 seconds)
	ShortTimeout = 5 * time.Second

	// LongTimeout is for operations that take longer (2 minutes)
	LongTimeout = 2 * time.Minute
)

// timeoutWriter wraps http.ResponseWriter to track if headers have been written
type timeoutWriter struct {
	http.ResponseWriter
	mu          sync.Mutex
	wroteHeader bool
	done        chan struct{}
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.wroteHeader {
		return
	}

	select {
	case <-tw.done:
		// Already timed out, don't write
		return
	default:
		tw.wroteHeader = true
		tw.ResponseWriter.WriteHeader(code)
	}
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	select {
	case <-tw.done:
		// Already timed out, don't write
		return 0, context.DeadlineExceeded
	default:
		if !tw.wroteHeader {
			tw.wroteHeader = true
			tw.ResponseWriter.WriteHeader(http.StatusOK)
		}
		return tw.ResponseWriter.Write(b)
	}
}
