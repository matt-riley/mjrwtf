package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/logging"
	"github.com/rs/zerolog"
)

// RequestID is a middleware that injects a request ID into the request context.
// If a X-Request-ID header is present, it uses that value; otherwise, it generates a new UUID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set the request ID in the response header
		w.Header().Set("X-Request-ID", requestID)

		// Add request ID to context
		ctx := logging.WithRequestID(r.Context(), requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// InjectLogger is a middleware that injects a logger with request-specific fields into the context.
func InjectLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := logging.GetRequestID(r.Context())

			// Create a logger with request-specific fields
			requestLogger := logger.With().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Logger()

			// Add logger to context
			ctx := logging.WithLogger(r.Context(), requestLogger)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
