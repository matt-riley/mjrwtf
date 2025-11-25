// Package logging provides structured logging functionality using zerolog.
package logging

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// LoggerKey is the context key for storing the logger.
	LoggerKey contextKey = "logger"
	// RequestIDKey is the context key for storing the request ID.
	RequestIDKey contextKey = "requestID"
)

// New creates a new zerolog logger with the specified configuration.
// level can be: debug, info, warn, error (default: info)
// format can be: json, pretty (default: json)
func New(level, format string) zerolog.Logger {
	var output io.Writer = os.Stdout

	// Set pretty format for development
	if format == "pretty" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	// Parse log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	return zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Logger()
}

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// FromContext retrieves the logger from the context.
// If no logger is found, returns a disabled logger.
func FromContext(ctx context.Context) zerolog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(zerolog.Logger); ok {
		return logger
	}
	return zerolog.Nop()
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
