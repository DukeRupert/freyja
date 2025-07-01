package middleware

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

func SetupLogger(levelStr, format string) zerolog.Logger {
	// Parse log level
	level, err := zerolog.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level '%s': %v\n", levelStr, err)
		fmt.Fprintf(os.Stderr, "Valid levels: trace, debug, info, warn, error, fatal, panic\n")
		os.Exit(1)
	}

	// Configure output format
	var logger zerolog.Logger
	if strings.ToLower(format) == "console" {
		// Pretty console output for development
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(output)
	} else {
		// JSON output for production
		logger = zerolog.New(os.Stdout)
	}

	return logger.
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()
}

// ZerologMiddleware creates Echo middleware that integrates with zerolog
func ZerologMiddleware(logger zerolog.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			// if user agent is prometheus, ignore
			agent := values.UserAgent
			if agent == "Prometheus/3.4.1" {
				return nil
			}

			// Generate request ID if not present
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = generateRequestID()
				c.Response().Header().Set(echo.HeaderXRequestID, requestID)
			}

			// Create request-scoped logger
			reqLogger := logger.With().
				Str("request_id", requestID).
				Str("method", values.Method).
				Str("uri", values.URI).
				Str("remote_ip", values.RemoteIP).
				Str("user_agent", values.UserAgent).
				Logger()

			// Store logger in context for handlers to use
			c.Set("logger", &reqLogger)

			// Log the request
			logEvent := reqLogger.Info()

			if values.Error != nil {
				logEvent = reqLogger.Error().Err(values.Error)
			}

			logEvent.
				Int("status", values.Status).
				Dur("latency", values.Latency).
				Msg("request processed")

			return nil
		},
	})
}

// Helper function to get logger from Echo context
func GetLogger(c echo.Context) *zerolog.Logger {
	if logger, ok := c.Get("logger").(*zerolog.Logger); ok {
		return logger
	}
	// Fallback to global zerolog logger
	globalLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &globalLogger
}

func generateRequestID() string {
	// Simple request ID generation - use uuid in production
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
