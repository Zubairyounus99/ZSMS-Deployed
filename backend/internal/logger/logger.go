package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
)

// New initializes a structured slog logger based on environment.
func New(env, serviceName string) *slog.Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: env == "development",
	}

	if strings.ToLower(env) == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		slog.String("service", serviceName),
		slog.String("env", env),
	)

	slog.SetDefault(logger)
	return logger
}

// WithRequestID adds a request ID attribute to the logger.
func WithRequestID(l *slog.Logger, reqID string) *slog.Logger {
	return l.With(slog.String("request_id", reqID))
}

// FromContext extracts request ID from context if available.
func FromContext(ctx context.Context, defaultLogger *slog.Logger) *slog.Logger {
	if ctx == nil {
		return defaultLogger
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		return defaultLogger.With(slog.String("request_id", reqID))
	}
	return defaultLogger
}
