package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiterConfig configures the rate limiter
type RateLimiterConfig struct {
	// RequestsPerSecond is the rate of token refill
	RequestsPerSecond float64

	// BurstSize is the maximum number of requests allowed in a burst
	BurstSize int

	// CleanupInterval is how often to clean up expired entries
	CleanupInterval time.Duration

	// KeyFunc extracts the rate limit key from the request
	// Default: client IP address
	KeyFunc func(r *http.Request) string
}

// DefaultRateLimiterConfig returns sensible defaults
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         20,
		CleanupInterval:   time.Minute,
		KeyFunc:           GetClientIP,
	}
}

// StrictRateLimiterConfig returns stricter limits for sensitive endpoints
func StrictRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerSecond: 1,
		BurstSize:         5,
		CleanupInterval:   time.Minute,
		KeyFunc:           GetClientIP,
	}
}

// tokenBucket implements a token bucket rate limiter
type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// RateLimiter is an in-memory rate limiter
type RateLimiter struct {
	config  RateLimiterConfig
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
	stop    chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.KeyFunc == nil {
		config.KeyFunc = GetClientIP
	}

	rl := &RateLimiter{
		config:  config,
		buckets: make(map[string]*tokenBucket),
		stop:    make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(rl.config.BurstSize),
			lastRefill: time.Now(),
		}
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * rl.config.RequestsPerSecond
	if bucket.tokens > float64(rl.config.BurstSize) {
		bucket.tokens = float64(rl.config.BurstSize)
	}
	bucket.lastRefill = now

	// Check if we have tokens available
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanup removes stale entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanupOnce()
		case <-rl.stop:
			return
		}
	}
}

// cleanupOnce performs a single cleanup pass.
// It collects stale keys first, then deletes them to avoid holding locks too long
// and to prevent potential issues with concurrent Allow() calls.
func (rl *RateLimiter) cleanupOnce() {
	now := time.Now()
	var staleKeys []string

	// First pass: identify stale buckets
	rl.mu.Lock()
	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		isStale := bucket.tokens >= float64(rl.config.BurstSize) &&
			now.Sub(bucket.lastRefill) > rl.config.CleanupInterval
		bucket.mu.Unlock()

		if isStale {
			staleKeys = append(staleKeys, key)
		}
	}
	rl.mu.Unlock()

	// Second pass: delete stale buckets
	if len(staleKeys) > 0 {
		rl.mu.Lock()
		for _, key := range staleKeys {
			// Re-check staleness in case the bucket was used between passes
			if bucket, exists := rl.buckets[key]; exists {
				bucket.mu.Lock()
				isStillStale := bucket.tokens >= float64(rl.config.BurstSize) &&
					now.Sub(bucket.lastRefill) > rl.config.CleanupInterval
				bucket.mu.Unlock()

				if isStillStale {
					delete(rl.buckets, key)
				}
			}
		}
		rl.mu.Unlock()
	}
}

// Stop stops the cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

// Middleware returns an HTTP middleware that applies rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.config.KeyFunc(r)

		if !rl.Allow(key) {
			w.Header().Set("Retry-After", "1")
			respondTooManyRequests(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimit creates a rate limiting middleware.
// If no config is provided, DefaultRateLimiterConfig is used.
func RateLimit(config ...RateLimiterConfig) func(http.Handler) http.Handler {
	var cfg RateLimiterConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultRateLimiterConfig()
	}
	limiter := NewRateLimiter(cfg)
	return limiter.Middleware
}

// StrictRateLimit creates a rate limiting middleware with stricter limits.
// Useful for sensitive endpoints like authentication.
func StrictRateLimit() func(http.Handler) http.Handler {
	return RateLimit(StrictRateLimiterConfig())
}

// GetClientIP extracts the client IP from the request
// It checks X-Forwarded-For and X-Real-IP headers first (for proxied requests)
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (comma-separated list, first is client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
