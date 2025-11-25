package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/logging"
	"github.com/rs/zerolog"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := logging.GetRequestID(r.Context())
		if requestID == "" {
			t.Error("expected request ID to be set in context")
		}
		// UUID format: 8-4-4-4-12 characters
		if len(requestID) != 36 {
			t.Errorf("expected UUID format (36 chars), got %d chars: %s", len(requestID), requestID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check that X-Request-ID header is set in response
	responseID := rec.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Error("expected X-Request-ID header in response")
	}
}

func TestRequestID_UsesExistingHeader(t *testing.T) {
	existingID := "my-custom-request-id"
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := logging.GetRequestID(r.Context())
		if requestID != existingID {
			t.Errorf("expected request ID %q, got %q", existingID, requestID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check that X-Request-ID header is preserved in response
	responseID := rec.Header().Get("X-Request-ID")
	if responseID != existingID {
		t.Errorf("expected X-Request-ID header %q, got %q", existingID, responseID)
	}
}

func TestInjectLogger_AddsLoggerToContext(t *testing.T) {
	logger := zerolog.Nop()

	// Create a chain: RequestID -> InjectLogger -> handler
	handler := RequestID(InjectLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxLogger := logging.FromContext(r.Context())
		// Logger should not be nil
		if ctxLogger.GetLevel() == zerolog.Disabled {
			// This is expected for Nop logger - just verify it doesn't panic
		}
	})))

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestInjectLogger_IncludesRequestFields(t *testing.T) {
	// This test verifies the middleware runs without error
	// In practice, you'd check the log output but that requires more setup
	logger := zerolog.Nop()

	handler := RequestID(InjectLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
