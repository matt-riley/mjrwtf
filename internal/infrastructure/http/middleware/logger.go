package middleware

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/logging"
	"github.com/rs/zerolog"
)

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	status      int
	size        int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(status int) {
	if !rw.wroteHeader {
		rw.status = status
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(status)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Flush implements http.Flusher
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Push implements http.Pusher
func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Logger logs HTTP requests with structured logging using zerolog.
// It logs method, path, status, response size, and duration.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// Get logger from context (or use a disabled logger if not present)
		logger := logging.FromContext(r.Context())

		// Determine log level based on status code
		var event *zerolog.Event
		switch {
		case wrapped.status >= 500:
			event = logger.Error()
		case wrapped.status >= 400:
			event = logger.Warn()
		default:
			event = logger.Info()
		}

		event.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapped.status).
			Int("size", wrapped.size).
			Dur("duration", duration).
			Msg("request completed")
	})
}
