package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPrometheusMetrics_RecordsRequest(t *testing.T) {
	m := metrics.New()

	handler := PrometheusMetrics(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify request was recorded
	count := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/health", "200"))
	if count != 1 {
		t.Errorf("expected 1 request recorded, got %f", count)
	}
}

func TestPrometheusMetrics_RecordsDuration(t *testing.T) {
	m := metrics.New()

	handler := PrometheusMetrics(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify histogram has observations
	count := testutil.CollectAndCount(m.HTTPRequestDuration)
	if count == 0 {
		t.Error("expected duration histogram to have observations")
	}
}

func TestPrometheusMetrics_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   string
	}{
		{"200 OK", http.StatusOK, "200"},
		{"404 Not Found", http.StatusNotFound, "404"},
		{"500 Server Error", http.StatusInternalServerError, "500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metrics.New()

			handler := PrometheusMetrics(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			count := testutil.ToFloat64(m.HTTPRequestsTotal.WithLabelValues("GET", "/health", tt.expected))
			if count != 1 {
				t.Errorf("expected 1 request with status %s, got %f", tt.expected, count)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"root path", "/", "/"},
		{"health endpoint", "/health", "/health"},
		{"metrics endpoint", "/metrics", "/metrics"},
		{"dashboard", "/dashboard", "/dashboard"},
		{"create page", "/create", "/create"},
		{"about page", "/about", "/about"},
		{"admin page", "/admin", "/admin"},
		{"login page", "/login", "/login"},
		{"logout page", "/logout", "/logout"},
		{"register page", "/register", "/register"},
		{"settings page", "/settings", "/settings"},
		{"favicon", "/favicon.ico", "/favicon.ico"},
		{"robots.txt", "/robots.txt", "/robots.txt"},
		{"API urls list", "/api/urls", "/api/urls"},
		{"API analytics path", "/api/urls/abc123/analytics", "/api/urls/{shortCode}/analytics"},
		{"API delete path", "/api/urls/abc123", "/api/urls/{shortCode}"},
		{"short code redirect", "/abc123", "/{shortCode}"},
		{"short code with dash", "/abc-123", "/{shortCode}"},
		{"short code with underscore", "/abc_123", "/{shortCode}"},
		{"short code mixed case", "/AbC123", "/{shortCode}"},
		{"path with slashes", "/some/nested/path", "/some/nested/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMetricsResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &metricsResponseWriter{ResponseWriter: rec, status: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)

	if rw.status != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rw.status)
	}

	// Second call should be ignored
	rw.WriteHeader(http.StatusBadRequest)
	if rw.status != http.StatusCreated {
		t.Errorf("expected status to remain %d, got %d", http.StatusCreated, rw.status)
	}
}

func TestMetricsResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &metricsResponseWriter{ResponseWriter: rec, status: http.StatusOK}

	n, err := rw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	// Check that WriteHeader was implicitly called with 200
	if !rw.wroteHeader {
		t.Error("expected wroteHeader to be true")
	}
	if rw.status != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rw.status)
	}
}
