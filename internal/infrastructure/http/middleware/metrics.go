package middleware

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/metrics"
)

// PrometheusMetrics returns a middleware that records HTTP request metrics
func PrometheusMetrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap the response writer to capture status code
			wrapped := &metricsResponseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(wrapped.status)

			// Normalize path to avoid high cardinality
			// Replace dynamic segments with placeholders
			path := normalizePath(r.URL.Path)

			m.RecordHTTPRequest(r.Method, path, status, duration)
		})
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code
type metricsResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *metricsResponseWriter) WriteHeader(status int) {
	if !rw.wroteHeader {
		rw.status = status
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(status)
	}
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher
func (rw *metricsResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker
func (rw *metricsResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Push implements http.Pusher
func (rw *metricsResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// normalizePath normalizes URL paths to reduce label cardinality
// It replaces dynamic segments (like short codes) with placeholders
func normalizePath(path string) string {
	// Handle common static paths - add new static routes here to prevent
	// them from being incorrectly treated as short codes
	switch path {
	case "/", "/health", "/ready", "/metrics", "/dashboard", "/create",
		"/about", "/admin", "/api", "/login", "/logout",
		"/register", "/settings", "/favicon.ico", "/robots.txt":
		return path
	}

	// Handle API paths
	if len(path) > 5 && path[:5] == "/api/" {
		// Keep API paths more granular but still normalized
		if len(path) > 10 && path[:10] == "/api/urls/" {
			// Replace short code in analytics path
			if len(path) > 20 && path[len(path)-10:] == "/analytics" {
				return "/api/urls/{shortCode}/analytics"
			}
			// Replace short code in delete path
			return "/api/urls/{shortCode}"
		}
		return path
	}

	// Root-level short code redirect (e.g., /abc123)
	if len(path) > 1 && path[0] == '/' {
		// Check if it looks like a short code (alphanumeric)
		isShortCode := true
		for _, c := range path[1:] {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				isShortCode = false
				break
			}
		}
		if isShortCode {
			return "/{shortCode}"
		}
	}

	return path
}
