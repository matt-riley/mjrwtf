package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/logging"
	"github.com/rs/zerolog"
)

func TestLogger_LogsRequest(t *testing.T) {
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "test response" {
		t.Errorf("expected response 'test response', got %s", rec.Body.String())
	}
}

func TestLogger_CapturesStatusCode(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{"status 200", http.StatusOK, http.StatusOK},
		{"status 404", http.StatusNotFound, http.StatusNotFound},
		{"status 500", http.StatusInternalServerError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestLogger_StructuredOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	// Add logger to context
	ctx := logging.WithLogger(req.Context(), logger)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Parse log output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("expected JSON log output, got: %s", buf.String())
	}

	// Verify log fields
	if logEntry["method"] != "GET" {
		t.Errorf("expected method GET, got %v", logEntry["method"])
	}
	if logEntry["path"] != "/api/test" {
		t.Errorf("expected path /api/test, got %v", logEntry["path"])
	}
	if logEntry["status"].(float64) != 200 {
		t.Errorf("expected status 200, got %v", logEntry["status"])
	}
	if logEntry["size"].(float64) != 5 {
		t.Errorf("expected size 5, got %v", logEntry["size"])
	}
	if _, ok := logEntry["duration"]; !ok {
		t.Error("expected duration field in log")
	}
}

func TestLogger_LogLevel_ByStatusCode(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel string
	}{
		{"2xx uses info", http.StatusOK, "info"},
		{"3xx uses info", http.StatusMovedPermanently, "info"},
		{"4xx uses warn", http.StatusNotFound, "warn"},
		{"5xx uses error", http.StatusInternalServerError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := zerolog.New(&buf)

			handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := logging.WithLogger(req.Context(), logger)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("expected JSON log output, got: %s", buf.String())
			}

			if logEntry["level"] != tt.expectedLevel {
				t.Errorf("expected level %s, got %v", tt.expectedLevel, logEntry["level"])
			}
		})
	}
}
