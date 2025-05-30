// internal/middleware/logger.go
package middleware

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

// RequestLogger returns a middleware that logs HTTP requests using zerolog
func RequestLogger(logger *zerolog.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:           true,
		LogStatus:        true,
		LogMethod:        true,
		LogRequestID:     true,
		LogLatency:       true,
		LogResponseSize:  true,
		LogUserAgent:     true,
		LogReferer:       true,
		LogRemoteIP:      true,
		LogContentLength: true,
		LogError:         true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// Get request ID from header or generate one
			reqID := c.Request().Header.Get(echo.HeaderXRequestID)
			if reqID == "" {
				reqID = uuid.New().String()
			}

			// Create log event with common fields
			logEvent := logger.Info()

			// Add request context
			logEvent.
				Str("req_id", reqID).
				Str("remote_ip", v.RemoteIP).
				Str("host", c.Request().Host).
				Str("method", v.Method).
				Str("uri", v.URI).
				Str("user_agent", v.UserAgent).
				Str("referer", v.Referer)

			// Add response information
			if v.Latency > 0 {
				logEvent.Dur("latency", v.Latency)
			}

			if v.Status > 0 {
				logEvent.Int("status", v.Status)

				// Add HTTP status category for easier filtering
				statusCategory := v.Status / 100
				logEvent.Int("status_category", statusCategory)
			}

			// Parse content length from string to int64
			if v.ContentLength != "" {
				if contentLength, err := strconv.ParseInt(v.ContentLength, 10, 64); err == nil && contentLength > 0 {
					logEvent.Int64("req_content_length", contentLength)
				}
			}

			if v.ResponseSize > 0 {
				logEvent.Int64("res_size", v.ResponseSize)
			}

			// Add route information
			if route := c.Path(); route != "" {
				logEvent.Str("route", route)
			}

			// Add user information if authenticated
			if user := c.Get("user"); user != nil {
				// This is just an example - adapt to your actual user type
				if u, ok := user.(map[string]interface{}); ok {
					if id, exists := u["id"]; exists {
						logEvent.Interface("user_id", id)
					}
				}
			}

			// Log any error
			if v.Error != nil {
				logEvent.Err(v.Error)

				// For 500 errors, we might want more details
				if v.Status >= 500 {
					logEvent.Stack()
				}

				logEvent.Msg("request error")
				return nil
			}

			// Determine log message based on status code
			msg := "request"
			if v.Status >= 400 && v.Status < 500 {
				msg = "request failed"
			} else if v.Status >= 500 {
				msg = "request error"
			} else if v.Status >= 300 && v.Status < 400 {
				msg = "request redirected"
			} else if v.Status >= 200 && v.Status < 300 {
				msg = "request successful"
			}

			logEvent.Msg(msg)
			return nil
		},
	})
}