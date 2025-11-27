package internal

import (
	"io"
	"log/slog"
	"time"
)

func NewLogger(w io.Writer, env string, level string) *slog.Logger {
	var h slog.Handler

	// Validate log level
	var l = new(slog.LevelVar) // Info by default
	switch level {
	case "debug":
		l.Set(slog.LevelDebug)
	case "warn":
		l.Set(slog.LevelWarn)
	case "error":
		l.Set(slog.LevelError)
	default:
		slog.Default().Warn("Invalid log level. Using default level: info", slog.String("value", level))
	}

	switch env {
	case "prod":
		h = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: l,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.String("time", a.Value.Time().Format(time.RFC3339Nano))
				}
				return a
			},
		})
	default:
		h = slog.NewTextHandler(w, &slog.HandlerOptions{Level: l})
	}

	return slog.New(h)
}
